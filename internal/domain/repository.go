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
}
