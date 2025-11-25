package dto

import (
	"errors"
	"strconv"
	"time"
)

type PaymentRequest struct {
	CustomerID           string `json:"customer_id"`
	PaymentStatus        string `json:"payment_status"`
	TransactionAmount    string `json:"transaction_amount"`
	TransactionDate      string `json:"transaction_date"`
	TransactionReference string `json:"transaction_reference"`
}

func (r *PaymentRequest) Validate() error {
	if r.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	if r.PaymentStatus == "" {
		return errors.New("payment_status is required")
	}
	if r.TransactionAmount == "" {
		return errors.New("transaction_amount is required")
	}
	if r.TransactionDate == "" {
		return errors.New("transaction_date is required")
	}
	if r.TransactionReference == "" {
		return errors.New("transaction_reference is required")
	}

	if _, err := strconv.ParseFloat(r.TransactionAmount, 64); err != nil {
		return errors.New("transaction_amount must be a valid number")
	}

	if _, err := time.Parse("2006-01-02 15:04:05", r.TransactionDate); err != nil {
		return errors.New("transaction_date must be in format 'YYYY-MM-DD HH:MM:SS'")
	}

	return nil
}

func (r *PaymentRequest) GetAmountInKobo() (int64, error) {
	amount, err := strconv.ParseFloat(r.TransactionAmount, 64)
	if err != nil {
		return 0, err
	}
	return int64(amount * 100), nil
}

func (r *PaymentRequest) GetTransactionDate() (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", r.TransactionDate)
}

type PaymentResponse struct {
	Success            bool    `json:"success"`
	Message            string  `json:"message"`
	CustomerID         string  `json:"customer_id,omitempty"`
	OutstandingBalance int64   `json:"outstanding_balance,omitempty"`
	TotalPaid          int64   `json:"total_paid,omitempty"`
	PaymentProgress    float64 `json:"payment_progress,omitempty"`
	IsFullyPaid        bool    `json:"is_fully_paid,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type CustomerResponse struct {
	CustomerID         string  `json:"customer_id"`
	AssetValue         int64   `json:"asset_value"`
	RepaymentTermWeeks int     `json:"repayment_term_weeks"`
	OutstandingBalance int64   `json:"outstanding_balance"`
	TotalPaid          int64   `json:"total_paid"`
	PaymentProgress    float64 `json:"payment_progress"`
	Status             string  `json:"status"`
	IsFullyPaid        bool    `json:"is_fully_paid"`
}
