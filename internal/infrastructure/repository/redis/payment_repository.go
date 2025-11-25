package redisrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/go-redis/redis/v8"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
)

type RedisPaymentRepository struct {
	client *redis.Client
}

func NewRedisPaymentRepository(client *redis.Client) *RedisPaymentRepository {
	return &RedisPaymentRepository{
		client: client,
	}
}

func (r *RedisPaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	key := r.paymentKey(payment.TransactionReference)

	data, err := json.Marshal(payment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	wasSet, err := r.client.SetNX(ctx, key, data, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to save payment: %w", err)
	}

	if !wasSet {
		return domain.ErrDuplicateTransaction
	}

	customerPaymentsKey := r.customerPaymentsKey(payment.CustomerID)
	if err := r.client.RPush(ctx, customerPaymentsKey, payment.TransactionReference).Err(); err != nil {
		return fmt.Errorf("failed to add payment to customer list: %w", err)
	}

	return nil
}

func (r *RedisPaymentRepository) FindByTransactionReference(ctx context.Context, txRef string) (*domain.Payment, error) {
	key := r.paymentKey(txRef)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	var payment domain.Payment
	if err := json.Unmarshal(data, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	return &payment, nil
}

func (r *RedisPaymentRepository) ExistsByTransactionReference(ctx context.Context, txRef string) (bool, error) {
	key := r.paymentKey(txRef)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check payment existence: %w", err)
	}

	return exists > 0, nil
}

func (r *RedisPaymentRepository) paymentKey(txRef string) string {
	return fmt.Sprintf("payment:%s", txRef)
}

func (r *RedisPaymentRepository) customerPaymentsKey(customerID string) string {
	return fmt.Sprintf("customer:%s:payments", customerID)
}
