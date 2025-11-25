package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type RedisEventSubscriber struct {
	client       *redis.Client
	logger       *zap.Logger
	handlers     map[string]domain.EventHandler
	consumerName string
	groupName    string
}

func NewRedisEventSubscriber(client *redis.Client, logger *zap.Logger, consumerName string) *RedisEventSubscriber {
	return &RedisEventSubscriber{
		client:       client,
		logger:       logger,
		handlers:     make(map[string]domain.EventHandler),
		consumerName: consumerName,
		groupName:    "payment-processors",
	}
}

func (s *RedisEventSubscriber) Subscribe(ctx context.Context, eventType string, handler domain.EventHandler) error {
	s.handlers[eventType] = handler

	streamKey := fmt.Sprintf("events:%s", eventType)

	// Create consumer group if doesn't exist
	err := s.client.XGroupCreateMkStream(ctx, streamKey, s.groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	s.logger.Info("subscribed to event",
		zap.String("event_type", eventType),
		zap.String("stream", streamKey),
		zap.String("group", s.groupName),
	)

	return nil
}

func (s *RedisEventSubscriber) Start(ctx context.Context) error {
	s.logger.Info("starting event subscriber",
		zap.String("consumer", s.consumerName),
		zap.String("group", s.groupName),
	)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping event subscriber")
			return nil
		default:
			if err := s.processEvents(ctx); err != nil {
				s.logger.Error("error processing events", zap.Error(err))
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (s *RedisEventSubscriber) processEvents(ctx context.Context) error {
	for eventType := range s.handlers {
		streamKey := fmt.Sprintf("events:%s", eventType)

		streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    s.groupName,
			Consumer: s.consumerName,
			Streams:  []string{streamKey, ">"},
			Count:    10,
			Block:    1 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			return fmt.Errorf("failed to read from stream: %w", err)
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := s.handleMessage(ctx, eventType, message); err != nil {
					s.logger.Error("failed to handle message",
						zap.Error(err),
						zap.String("message_id", message.ID),
						zap.String("stream", streamKey),
					)
					continue
				}

				s.client.XAck(ctx, streamKey, s.groupName, message.ID)
			}
		}
	}

	return nil
}

func (s *RedisEventSubscriber) handleMessage(ctx context.Context, eventType string, message redis.XMessage) error {
	handler, exists := s.handlers[eventType]
	if !exists {
		return fmt.Errorf("no handler for event type: %s", eventType)
	}

	eventData, ok := message.Values["data"].(string)
	if !ok {
		return fmt.Errorf("invalid event data format")
	}

	var event domain.DomainEvent
	switch eventType {
	case domain.EventTypePaymentProcessed:
		var e domain.PaymentProcessedEvent
		if err := json.Unmarshal([]byte(eventData), &e); err != nil {
			return fmt.Errorf("failed to unmarshal event: %w", err)
		}
		event = &e
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	return handler(ctx, event)
}
