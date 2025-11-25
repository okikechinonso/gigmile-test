package domain

import "context"

type CustomerRepository interface {
	FindByID(ctx context.Context, customerID string) (*Customer, error)
	Save(ctx context.Context, customer *Customer) error
	UpdateBalance(ctx context.Context, customerID string, amount int64, version int64) error
}

type PaymentRepository interface {
	Save(ctx context.Context, payment *Payment) error
	FindByTransactionReference(ctx context.Context, txRef string) (*Payment, error)
	ExistsByTransactionReference(ctx context.Context, txRef string) (bool, error)
	FindByCustomerID(ctx context.Context, customerID string) ([]*Payment, error)
	FindByCustomerIDWithPagination(ctx context.Context, customerID string, limit, offset int) ([]*Payment, error)
	CountByCustomerID(ctx context.Context, customerID string) (int64, error)
}
