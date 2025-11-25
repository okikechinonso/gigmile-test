package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type RedisEventPublisher struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisEventPublisher(client *redis.Client, logger *zap.Logger) *RedisEventPublisher {
	return &RedisEventPublisher{
		client: client,
		logger: logger,
	}
}

func (p *RedisEventPublisher) Publish(ctx context.Context, event domain.DomainEvent) error {
	streamKey := fmt.Sprintf("events:%s", event.GetEventType())

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: streamKey,
		MaxLen: 100000, // Keep last 100k events
		Approx: true,
		Values: map[string]interface{}{
			"event_id":     event.GetEventID(),
			"event_type":   event.GetEventType(),
			"aggregate_id": event.GetAggregateID(),
			"occurred_at":  event.GetOccurredAt().Unix(),
			"data":         string(eventData),
		},
	}

	_, err = p.client.XAdd(ctx, args).Result()
	if err != nil {
		p.logger.Error("failed to publish event",
			zap.Error(err),
			zap.String("event_type", event.GetEventType()),
			zap.String("event_id", event.GetEventID()),
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Debug("event published",
		zap.String("event_type", event.GetEventType()),
		zap.String("event_id", event.GetEventID()),
		zap.String("stream", streamKey),
	)

	return nil
}
