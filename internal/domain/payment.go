package domain

import (
	"errors"
	"time"
)

type Payment struct {
	ID                   string
	CustomerID           string
	Amount               int64 // in kobo
	TransactionReference string
	TransactionDate      time.Time
	Status               PaymentStatus
	ProcessedAt          time.Time
	CreatedAt            time.Time
}

var ErrOptimisticLock = errors.New("version mismatch - optimistic lock failed")

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusComplete  PaymentStatus = "COMPLETE"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusDuplicate PaymentStatus = "DUPLICATE"
)

func NewPayment(customerID string, amount int64, transactionRef string, transactionDate time.Time, status PaymentStatus) (*Payment, error) {
	if customerID == "" {
		return nil, ErrInvalidCustomerID
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if transactionRef == "" {
		return nil, ErrInvalidTransactionRef
	}

	now := time.Now()
	return &Payment{
		CustomerID:           customerID,
		Amount:               amount,
		TransactionReference: transactionRef,
		TransactionDate:      transactionDate,
		Status:               status,
		CreatedAt:            now,
	}, nil
}

func (p *Payment) MarkAsProcessed() {
	p.ProcessedAt = time.Now()
}

func (p *Payment) IsDuplicate() bool {
	return p.Status == PaymentStatusDuplicate
}
