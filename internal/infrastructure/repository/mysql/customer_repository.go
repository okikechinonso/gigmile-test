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
	// Try Redis cache first for high throughput
	cached, err := r.redisRepo.FindByID(ctx, id)
	if err == nil {
		r.logger.Debug("customer cache hit", zap.String("customer_id", id))
		return cached, nil
	}

	// Cache miss - query MySQL
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

	// Update cache asynchronously
	go r.redisRepo.Save(context.Background(), customer)

	return customer, nil
}

func (r *GORMCustomerRepository) Save(ctx context.Context, customer *domain.Customer) error {
	model := persistence.CustomerModelFromDomain(customer)

	// CRITICAL: Invalidate cache BEFORE updating MySQL
	// This ensures concurrent requests will fetch fresh data from MySQL
	if err := r.redisRepo.Delete(ctx, customer.ID); err != nil {
		r.logger.Warn("failed to invalidate cache before save", 
			zap.Error(err), 
			zap.String("customer_id", customer.ID))
	}

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
		return domain.ErrOptimisticLock // Concurrent modification detected
	}

	// Update version (GORM auto-incremented it)
	customer.Version++

	// Update cache with latest data
	if err := r.redisRepo.Save(ctx, customer); err != nil {
		r.logger.Warn("failed to update cache after save",
			zap.Error(err),
			zap.String("customer_id", customer.ID))
	}

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

	return nil
}

func (r *GORMCustomerRepository) UpdateBalance(ctx context.Context, customerID string, amount int64, version int64) error {
	// Invalidate cache before update
	if err := r.redisRepo.Delete(ctx, customerID); err != nil {
		r.logger.Warn("failed to invalidate cache before balance update", zap.Error(err))
	}

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
		return domain.ErrOptimisticLock
	}

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
