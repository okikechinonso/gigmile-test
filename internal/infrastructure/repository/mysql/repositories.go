package sqlrepository

import (
	"errors"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrPaymentNotFound  = errors.New("payment not found")
)

type Repositories struct {
	Customer domain.CustomerRepository
	Payment  domain.PaymentRepository
}

func NewRepositories(db *gorm.DB, redisClient *redis.Client, logger *zap.Logger) *Repositories {
	return &Repositories{
		Customer: NewCustomerRepository(db, redisClient, logger),
		Payment:  NewPaymentRepository(db, redisClient, logger),
	}
}
