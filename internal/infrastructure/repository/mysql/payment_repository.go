package sqlrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/gigmile/payment-service/internal/infrastructure/persistence"
	redisrepository "github.com/gigmile/payment-service/internal/infrastructure/repository/redis"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type GORMPaymentRepository struct {
	db        *gorm.DB
	redisRepo *redisrepository.RedisPaymentRepository
	logger    *zap.Logger
}

func NewPaymentRepository(db *gorm.DB, redisClient *redis.Client, logger *zap.Logger) *GORMPaymentRepository {
	return &GORMPaymentRepository{
		db:        db,
		redisRepo: redisrepository.NewRedisPaymentRepository(redisClient),
		logger:    logger,
	}
}

func (r *GORMPaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	// 1. Fast idempotency check in Redis (before hitting MySQL)
	exists, err := r.redisRepo.ExistsByTransactionReference(ctx, payment.TransactionReference)
	if err != nil {
		r.logger.Warn("redis dedup check failed, falling back to MySQL", zap.Error(err))
	} else if exists {
		// Already exists in Redis
		return domain.ErrDuplicateTransaction
	}

	// 2. Generate ID if not set
	if payment.ID == "" {
		payment.ID = uuid.New().String()
	}

	// 3. Insert into MySQL using GORM
	model := persistence.PaymentModelFromDomain(payment)

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		// Check if duplicate key error
		if isDuplicateError(result.Error) {
			return domain.ErrDuplicateTransaction
		}

		r.logger.Error("failed to save payment", zap.Error(result.Error))
		return fmt.Errorf("database error: %w", result.Error)
	}

	r.logger.Debug("payment saved to MySQL",
		zap.String("payment_id", payment.ID),
		zap.String("tx_ref", payment.TransactionReference),
	)

	return nil
}

func (r *GORMPaymentRepository) FindByTransactionReference(ctx context.Context, txRef string) (*domain.Payment, error) {
	var model persistence.PaymentModel

	result := r.db.WithContext(ctx).
		Where("transaction_reference = ?", txRef).
		First(&model)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	payment := model.ToDomain()

	// Cache in Redis
	go r.redisRepo.Save(context.Background(), payment)

	return payment, nil
}

func (r *GORMPaymentRepository) ExistsByTransactionReference(ctx context.Context, txRef string) (bool, error) {
	// 1. Check Redis first (fast path)
	exists, err := r.redisRepo.ExistsByTransactionReference(ctx, txRef)
	if err == nil && exists {
		r.logger.Debug("payment exists (Redis cache)", zap.String("tx_ref", txRef))
		return true, nil
	}

	// 2. Check MySQL (slower but source of truth)
	var count int64
	result := r.db.WithContext(ctx).
		Model(&persistence.PaymentModel{}).
		Where("transaction_reference = ?", txRef).
		Count(&count)

	if result.Error != nil {
		r.logger.Error("failed to check payment existence", zap.Error(result.Error))
		return false, fmt.Errorf("database error: %w", result.Error)
	}

	existsInDB := count > 0

	// 3. If exists in MySQL, update Redis cache
	if existsInDB {
		// Fetch the payment to cache it
		payment, err := r.FindByTransactionReference(ctx, txRef)
		if err == nil {
			go r.redisRepo.Save(context.Background(), payment)
		}
	}

	return existsInDB, nil
}

func (r *GORMPaymentRepository) FindByCustomerID(ctx context.Context, customerID string) ([]*domain.Payment, error) {
	var models []persistence.PaymentModel

	result := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("transaction_date DESC").
		Find(&models)

	if result.Error != nil {
		r.logger.Error("failed to fetch payments by customer ID",
			zap.Error(result.Error),
			zap.String("customer_id", customerID),
		)
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	payments := make([]*domain.Payment, len(models))
	for i, model := range models {
		payments[i] = model.ToDomain()
	}

	r.logger.Debug("fetched payments by customer ID",
		zap.String("customer_id", customerID),
		zap.Int("count", len(payments)),
	)

	return payments, nil
}

func (r *GORMPaymentRepository) GetTotalPaidByCustomer(ctx context.Context, customerID string) (int64, error) {
	var total int64

	result := r.db.WithContext(ctx).
		Model(&persistence.PaymentModel{}).
		Where("customer_id = ? AND status = ?", customerID, "COMPLETE").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to calculate total: %w", result.Error)
	}

	return total, nil
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		(err.Error() != "" && (contains(err.Error(), "Duplicate entry") ||
			contains(err.Error(), "UNIQUE constraint")))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
