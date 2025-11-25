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
	exists, err := r.redisRepo.ExistsByTransactionReference(ctx, payment.TransactionReference)
	if err != nil {
		r.logger.Warn("redis dedup check failed, falling back to MySQL", zap.Error(err))
	} else if exists {
		return domain.ErrDuplicateTransaction
	}

	if payment.ID == "" {
		payment.ID = uuid.New().String()
	}

	model := persistence.PaymentModelFromDomain(payment)

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
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
	exists, err := r.redisRepo.ExistsByTransactionReference(ctx, txRef)
	if err == nil && exists {
		r.logger.Debug("payment exists (Redis cache)", zap.String("tx_ref", txRef))
		return true, nil
	}

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

	if existsInDB {
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

func (r *GORMPaymentRepository) FindByCustomerIDWithPagination(ctx context.Context, customerID string, limit, offset int) ([]*domain.Payment, error) {
	var models []persistence.PaymentModel

	result := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("transaction_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&models)

	if result.Error != nil {
		r.logger.Error("failed to fetch payments by customer ID with pagination",
			zap.Error(result.Error),
			zap.String("customer_id", customerID),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
		)
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	payments := make([]*domain.Payment, len(models))
	for i, model := range models {
		payments[i] = model.ToDomain()
	}

	r.logger.Debug("fetched payments by customer ID with pagination",
		zap.String("customer_id", customerID),
		zap.Int("count", len(payments)),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	return payments, nil
}

func (r *GORMPaymentRepository) CountByCustomerID(ctx context.Context, customerID string) (int64, error) {
	var count int64

	result := r.db.WithContext(ctx).
		Model(&persistence.PaymentModel{}).
		Where("customer_id = ?", customerID).
		Count(&count)

	if result.Error != nil {
		r.logger.Error("failed to count payments by customer ID",
			zap.Error(result.Error),
			zap.String("customer_id", customerID),
		)
		return 0, fmt.Errorf("database error: %w", result.Error)
	}

	r.logger.Debug("counted payments by customer ID",
		zap.String("customer_id", customerID),
		zap.Int64("count", count),
	)

	return count, nil
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
