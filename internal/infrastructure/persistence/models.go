package persistence

import (
	"time"

	"github.com/gigmile/payment-service/internal/domain"
)

// CustomerModel represents the database schema for customers
type CustomerModel struct {
	ID                 string    `gorm:"primaryKey;type:varchar(50)"`
	AssetValue         int64     `gorm:"not null"`
	OutstandingBalance int64     `gorm:"not null"`
	TotalPaid          int64     `gorm:"not null;default:0"`
	RepaymentTermWeeks int       `gorm:"not null"`
	DeploymentDate     time.Time `gorm:"not null"`
	LastPaymentDate    *time.Time
	Status             string    `gorm:"type:varchar(20);not null;index"`
	Version            int64     `gorm:"not null;default:1"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}

func (CustomerModel) TableName() string {
	return "customers"
}

// ToDomain converts database model to domain entity
func (m *CustomerModel) ToDomain() *domain.Customer {
	return &domain.Customer{
		ID:                 m.ID,
		AssetValue:         m.AssetValue,
		OutstandingBalance: m.OutstandingBalance,
		TotalPaid:          m.TotalPaid,
		RepaymentTermWeeks: m.RepaymentTermWeeks,
		DeploymentDate:     m.DeploymentDate,
		LastPaymentDate:    m.LastPaymentDate,
		Status:             domain.CustomerStatus(m.Status),
		Version:            m.Version,
	}
}

// FromDomain converts domain entity to database model
func CustomerModelFromDomain(customer *domain.Customer) *CustomerModel {
	return &CustomerModel{
		ID:                 customer.ID,
		AssetValue:         customer.AssetValue,
		OutstandingBalance: customer.OutstandingBalance,
		TotalPaid:          customer.TotalPaid,
		RepaymentTermWeeks: customer.RepaymentTermWeeks,
		DeploymentDate:     customer.DeploymentDate,
		LastPaymentDate:    customer.LastPaymentDate,
		Status:             string(customer.Status),
		Version:            customer.Version,
	}
}

// PaymentModel represents the database schema for payments
type PaymentModel struct {
	ID                   string     `gorm:"primaryKey;type:varchar(50)"`
	CustomerID           string     `gorm:"type:varchar(50);not null;index"`
	Amount               int64      `gorm:"not null"`
	TransactionReference string     `gorm:"type:varchar(100);uniqueIndex;not null"`
	TransactionDate      time.Time  `gorm:"not null;index"`
	Status               string     `gorm:"type:varchar(20);not null"`
	ProcessedAt          *time.Time `gorm:"index"`
	CreatedAt            time.Time  `gorm:"autoCreateTime"`
}

func (PaymentModel) TableName() string {
	return "payments"
}

// ToDomain converts database model to domain entity
func (m *PaymentModel) ToDomain() *domain.Payment {
	payment := &domain.Payment{
		ID:                   m.ID,
		CustomerID:           m.CustomerID,
		Amount:               m.Amount,
		TransactionReference: m.TransactionReference,
		TransactionDate:      m.TransactionDate,
		Status:               domain.PaymentStatus(m.Status),
		CreatedAt:            m.CreatedAt,
	}
	if m.ProcessedAt != nil {
		payment.ProcessedAt = *m.ProcessedAt
	}
	return payment
}

// FromDomain converts domain entity to database model
func PaymentModelFromDomain(payment *domain.Payment) *PaymentModel {
	model := &PaymentModel{
		ID:                   payment.ID,
		CustomerID:           payment.CustomerID,
		Amount:               payment.Amount,
		TransactionReference: payment.TransactionReference,
		TransactionDate:      payment.TransactionDate,
		Status:               string(payment.Status),
		CreatedAt:            payment.CreatedAt,
	}
	if !payment.ProcessedAt.IsZero() {
		model.ProcessedAt = &payment.ProcessedAt
	}
	return model
}
