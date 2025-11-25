package redisrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/go-redis/redis/v8"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrVersionMismatch  = errors.New("version mismatch - optimistic lock failed")
)

type RedisCustomerRepository struct {
	client   *redis.Client
	cacheTTL time.Duration
}

func NewRedisCustomerRepository(client *redis.Client, cacheTTL time.Duration) *RedisCustomerRepository {
	return &RedisCustomerRepository{
		client:   client,
		cacheTTL: cacheTTL,
	}
}

func (r *RedisCustomerRepository) FindByID(ctx context.Context, customerID string) (*domain.Customer, error) {
	key := r.customerKey(customerID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	var customer domain.Customer
	if err := json.Unmarshal(data, &customer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal customer: %w", err)
	}

	return &customer, nil
}

func (r *RedisCustomerRepository) Save(ctx context.Context, customer *domain.Customer) error {
	key := r.customerKey(customer.ID)

	data, err := json.Marshal(customer)
	if err != nil {
		return fmt.Errorf("failed to marshal customer: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.cacheTTL).Err(); err != nil {
		return fmt.Errorf("failed to save customer: %w", err)
	}

	return nil
}

func (r *RedisCustomerRepository) UpdateBalance(ctx context.Context, customerID string, amount int64, version int64) error {
	script := `
		local key = KEYS[1]
		local amount = tonumber(ARGV[1])
		local expected_version = tonumber(ARGV[2])
		
		local data = redis.call('GET', key)
		if not data then
			return redis.error_reply('customer not found')
		end
		
		local customer = cjson.decode(data)
		
		if customer.Version ~= expected_version then
			return redis.error_reply('version mismatch')
		end
		
		customer.OutstandingBalance = customer.OutstandingBalance - amount
		if customer.OutstandingBalance < 0 then
			customer.OutstandingBalance = 0
		end
		
		customer.TotalPaid = customer.TotalPaid + amount
		customer.LastPaymentDate = ARGV[3]
		customer.Version = customer.Version + 1
		
		if customer.OutstandingBalance == 0 then
			customer.Status = 'COMPLETED'
		end
		
		redis.call('SET', key, cjson.encode(customer))
		return 'OK'
	`

	key := r.customerKey(customerID)
	now := time.Now().Format(time.RFC3339)

	result, err := r.client.Eval(ctx, script, []string{key}, amount, version, now).Result()
	if err != nil {
		if err.Error() == "customer not found" {
			return ErrCustomerNotFound
		}
		if err.Error() == "version mismatch" {
			return ErrVersionMismatch
		}
		return fmt.Errorf("failed to update balance: %w", err)
	}

	if result != "OK" {
		return errors.New("unexpected result from update")
	}

	return nil
}

func (r *RedisCustomerRepository) customerKey(customerID string) string {
	return fmt.Sprintf("customer:%s", customerID)
}

func (r *RedisCustomerRepository) Delete(ctx context.Context, customerID string) error {
	key := r.customerKey(customerID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}
	return nil
}
