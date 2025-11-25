package domain

import (
	"errors"
	"time"
)

// Domain errors
var (
	ErrInvalidCustomerID     = errors.New("invalid customer ID")
	ErrInvalidAmount         = errors.New("invalid transaction amount")
	ErrInvalidTransactionRef = errors.New("invalid transaction reference")
	ErrDuplicateTransaction  = errors.New("duplicate transaction")
	ErrInsufficientBalance   = errors.New("insufficient balance for operation")
	ErrAssetAlreadyOwned     = errors.New("asset already fully owned")
)

// Customer represents the aggregate root in DDD
type Customer struct {
	ID                 string
	AssetValue         int64 // in kobo (N1,000,000 = 100,000,000 kobo)
	RepaymentTermWeeks int
	OutstandingBalance int64
	TotalPaid          int64
	DeploymentDate     time.Time
	LastPaymentDate    *time.Time
	Status             CustomerStatus
	Version            int64 // for optimistic locking
}

type CustomerStatus string

const (
	CustomerStatusActive    CustomerStatus = "ACTIVE"
	CustomerStatusCompleted CustomerStatus = "COMPLETED"
	CustomerStatusDefaulted CustomerStatus = "DEFAULTED"
)

// NewCustomer creates a new customer with asset deployment
func NewCustomer(id string, assetValue int64, termWeeks int, deploymentDate time.Time) (*Customer, error) {
	if id == "" {
		return nil, ErrInvalidCustomerID
	}
	if assetValue <= 0 {
		return nil, errors.New("asset value must be positive")
	}
	if termWeeks <= 0 {
		return nil, errors.New("repayment term must be positive")
	}

	return &Customer{
		ID:                 id,
		AssetValue:         assetValue,
		RepaymentTermWeeks: termWeeks,
		OutstandingBalance: assetValue,
		TotalPaid:          0,
		DeploymentDate:     deploymentDate,
		Status:             CustomerStatusActive,
		Version:            1,
	}, nil
}

// ApplyPayment applies a payment to the customer's account
func (c *Customer) ApplyPayment(amount int64, paymentDate time.Time) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if c.Status == CustomerStatusCompleted {
		return ErrAssetAlreadyOwned
	}

	// Calculate new balance
	newBalance := c.OutstandingBalance - amount
	if newBalance < 0 {
		// Overpayment: set balance to 0
		newBalance = 0
	}

	c.OutstandingBalance = newBalance
	c.TotalPaid += amount
	c.LastPaymentDate = &paymentDate

	// Mark as completed if fully paid
	if c.OutstandingBalance == 0 {
		c.Status = CustomerStatusCompleted
	}

	// Note: Version is incremented by the repository during persistence
	return nil
}

// GetPaymentProgress returns the percentage of asset paid
func (c *Customer) GetPaymentProgress() float64 {
	if c.AssetValue == 0 {
		return 0
	}
	return float64(c.TotalPaid) / float64(c.AssetValue) * 100
}

// IsFullyPaid checks if the customer has fully paid for the asset
func (c *Customer) IsFullyPaid() bool {
	return c.OutstandingBalance == 0 || c.Status == CustomerStatusCompleted
}
