package sqlrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/gigmile/payment-service/internal/infrastructure/persistence"
	redisrepository "github.com/gigmile/payment-service/internal/infrastructure/repository/redis"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type GORMCustomerRepository struct {
	db        *gorm.DB
	redisRepo *redisrepository.RedisCustomerRepository
	logger    *zap.Logger
}

func NewCustomerRepository(db *gorm.DB, redisClient *redis.Client, logger *zap.Logger) *GORMCustomerRepository {
	return &GORMCustomerRepository{
		db:        db,
		redisRepo: redisrepository.NewRedisCustomerRepository(redisClient, 5*time.Minute),
		logger:    logger,
	}
}

func (r *GORMCustomerRepository) FindByID(ctx context.Context, id string) (*domain.Customer, error) {
	// 1. Try Redis cache first (hot path)
	cached, err := r.redisRepo.FindByID(ctx, id)
	if err == nil {
		r.logger.Debug("customer cache hit", zap.String("customer_id", id))
		return cached, nil
	}

	// 2. Cache miss - query MySQL with GORM
	r.logger.Debug("customer cache miss, querying MySQL", zap.String("customer_id", id))

	var model persistence.CustomerModel
	result := r.db.WithContext(ctx).First(&model, "id = ?", id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		r.logger.Error("failed to query customer", zap.Error(result.Error))
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	customer := model.ToDomain()

	// 3. Update Redis cache (async - don't block)
	go r.redisRepo.Save(context.Background(), customer)

	return customer, nil
}

func (r *GORMCustomerRepository) Save(ctx context.Context, customer *domain.Customer) error {
	model := persistence.CustomerModelFromDomain(customer)

	// Use optimistic locking with GORM
	result := r.db.WithContext(ctx).
		Model(&persistence.CustomerModel{}).
		Where("id = ? AND version = ?", customer.ID, customer.Version).
		Updates(map[string]interface{}{
			"outstanding_balance": model.OutstandingBalance,
			"total_paid":          model.TotalPaid,
			"last_payment_date":   model.LastPaymentDate,
			"status":              model.Status,
			"version":             gorm.Expr("version + 1"),
			"updated_at":          time.Now(),
		})

	if result.Error != nil {
		r.logger.Error("failed to update customer", zap.Error(result.Error))
		return fmt.Errorf("database error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrVersionMismatch // Concurrent modification detected
	}

	// Update version (GORM auto-incremented it)
	customer.Version++

	// Invalidate cache (write-through)
	r.redisRepo.Delete(ctx, customer.ID)

	// Update cache with new data (async)
	go r.redisRepo.Save(context.Background(), customer)

	r.logger.Debug("customer saved to MySQL",
		zap.String("customer_id", customer.ID),
		zap.Int64("version", customer.Version),
	)

	return nil
}

func (r *GORMCustomerRepository) Create(ctx context.Context, customer *domain.Customer) error {
	model := persistence.CustomerModelFromDomain(customer)

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		r.logger.Error("failed to create customer", zap.Error(result.Error))
		return fmt.Errorf("failed to create customer: %w", result.Error)
	}

	r.logger.Info("customer created",
		zap.String("customer_id", customer.ID),
	)

	// Cache the new customer
	go r.redisRepo.Save(context.Background(), customer)

	return nil
}

func (r *GORMCustomerRepository) UpdateBalance(ctx context.Context, customerID string, amount int64, version int64) error {
	// This is an alternative to Save() for simpler balance updates
	result := r.db.WithContext(ctx).
		Model(&persistence.CustomerModel{}).
		Where("id = ? AND version = ?", customerID, version).
		Updates(map[string]interface{}{
			"outstanding_balance": gorm.Expr("outstanding_balance - ?", amount),
			"total_paid":          gorm.Expr("total_paid + ?", amount),
			"version":             gorm.Expr("version + 1"),
			"updated_at":          time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update balance: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrVersionMismatch
	}

	// Invalidate cache
	r.redisRepo.Delete(ctx, customerID)

	return nil
}

func (r *GORMCustomerRepository) FindByStatus(ctx context.Context, status string, limit int) ([]*domain.Customer, error) {
	var models []persistence.CustomerModel

	result := r.db.WithContext(ctx).
		Where("status = ?", status).
		Limit(limit).
		Order("created_at DESC").
		Find(&models)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query customers: %w", result.Error)
	}

	customers := make([]*domain.Customer, len(models))
	for i, model := range models {
		customers[i] = model.ToDomain()
	}

	return customers, nil
}
