package main

// ...existing code...
import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gigmile/payment-service/internal/application/service"
	"github.com/gigmile/payment-service/internal/config"
	"github.com/gigmile/payment-service/internal/domain"
	"github.com/gigmile/payment-service/internal/infrastructure/messaging"
	"github.com/gigmile/payment-service/internal/infrastructure/repository/redis"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// ...existing code...

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Load configuration
	cfg := config.Load()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	logger.Info("connected to Redis successfully")

	// Initialize repositories
	customerRepo := redisrepository.NewRedisCustomerRepository(redisClient, 0)

	// Initialize notification service
	notificationService := service.NewNotificationService(customerRepo, logger)

	// Initialize event subscriber
	hostname, _ := os.Hostname()
	consumerName := fmt.Sprintf("worker-%s-%d", hostname, os.Getpid())
	eventSubscriber := messaging.NewRedisEventSubscriber(redisClient, logger, consumerName)

	// Subscribe to payment processed events
	if err := eventSubscriber.Subscribe(ctx, domain.EventTypePaymentProcessed, notificationService.HandlePaymentProcessed); err != nil {
		logger.Fatal("failed to subscribe to events", zap.Error(err))
	}

	logger.Info("worker started",
		zap.String("consumer", consumerName),
		zap.String("event_type", domain.EventTypePaymentProcessed),
	)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutting down worker...")
		cancel()
	}()

	// Start processing events
	if err := eventSubscriber.Start(ctx); err != nil {
		logger.Info("worker stopped", zap.Error(err))
	}

	logger.Info("worker exited")
}
