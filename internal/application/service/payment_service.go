package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gigmile/payment-service/internal/domain"
	"go.uber.org/zap"
)

type PaymentService struct {
	customerRepo   domain.CustomerRepository
	paymentRepo    domain.PaymentRepository
	eventPublisher domain.EventPublisher // Optional - can be nil
	logger         *zap.Logger
}

// NewPaymentService creates a payment service with event publishing
func NewPaymentService(
	customerRepo domain.CustomerRepository,
	paymentRepo domain.PaymentRepository,
	eventPublisher domain.EventPublisher,
	logger *zap.Logger,
) *PaymentService {
	return &PaymentService{
		customerRepo:   customerRepo,
		paymentRepo:    paymentRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

type ProcessPaymentRequest struct {
	CustomerID           string
	PaymentStatus        string
	TransactionAmount    int64
	TransactionDate      time.Time
	TransactionReference string
}

type ProcessPaymentResponse struct {
	Success            bool
	Message            string
	CustomerID         string
	OutstandingBalance int64
	TotalPaid          int64
	PaymentProgress    float64
	IsFullyPaid        bool
}

func (s *PaymentService) ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (*ProcessPaymentResponse, error) {
	if req.PaymentStatus != "COMPLETE" {
		s.logger.Info("payment not complete",
			zap.String("customer_id", req.CustomerID),
			zap.String("status", req.PaymentStatus),
			zap.String("tx_ref", req.TransactionReference),
		)
		return &ProcessPaymentResponse{
			Success: false,
			Message: fmt.Sprintf("payment status is %s, not COMPLETE", req.PaymentStatus),
		}, nil
	}

	exists, err := s.paymentRepo.ExistsByTransactionReference(ctx, req.TransactionReference)
	if err != nil {
		s.logger.Error("failed to check payment existence",
			zap.Error(err),
			zap.String("tx_ref", req.TransactionReference),
		)
		return nil, fmt.Errorf("failed to check payment existence: %w", err)
	}

	if exists {
		s.logger.Info("duplicate payment detected",
			zap.String("customer_id", req.CustomerID),
			zap.String("tx_ref", req.TransactionReference),
		)

		customer, err := s.customerRepo.FindByID(ctx, req.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer for duplicate payment: %w", err)
		}

		return &ProcessPaymentResponse{
			Success:            true,
			Message:            "duplicate transaction - already processed",
			CustomerID:         customer.ID,
			OutstandingBalance: customer.OutstandingBalance,
			TotalPaid:          customer.TotalPaid,
			PaymentProgress:    customer.GetPaymentProgress(),
			IsFullyPaid:        customer.IsFullyPaid(),
		}, nil
	}

	// Get customer BEFORE creating/saving payment
	customer, err := s.customerRepo.FindByID(ctx, req.CustomerID)
	if err != nil {
		s.logger.Error("failed to get customer",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID),
		)
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Apply payment to customer entity
	if err := customer.ApplyPayment(req.TransactionAmount, req.TransactionDate); err != nil {
		s.logger.Error("failed to apply payment",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID),
		)
		return nil, fmt.Errorf("failed to apply payment: %w", err)
	}

	// Save customer with updated balance FIRST
	// Retry once on optimistic lock failure
	err = s.customerRepo.Save(ctx, customer)
	if err == domain.ErrOptimisticLock {
		s.logger.Warn("optimistic lock conflict, retrying once",
			zap.String("customer_id", req.CustomerID),
		)

		// Refetch and retry
		customer, err = s.customerRepo.FindByID(ctx, req.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer on retry: %w", err)
		}

		if err := customer.ApplyPayment(req.TransactionAmount, req.TransactionDate); err != nil {
			return nil, fmt.Errorf("failed to apply payment on retry: %w", err)
		}

		err = s.customerRepo.Save(ctx, customer)
	}

	if err != nil {
		s.logger.Error("failed to save customer",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID),
		)
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	// Now save the payment record AFTER customer is successfully updated
	payment, err := domain.NewPayment(
		req.CustomerID,
		req.TransactionAmount,
		req.TransactionReference,
		req.TransactionDate,
		domain.PaymentStatusComplete,
	)
	if err != nil {
		s.logger.Error("failed to create payment entity",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID),
		)
		return nil, fmt.Errorf("invalid payment data: %w", err)
	}

	if err := s.paymentRepo.Save(ctx, payment); err != nil {
		if err == domain.ErrDuplicateTransaction {
			// Payment already exists, but customer was updated
			// This is a race condition scenario - return success
			s.logger.Warn("duplicate payment detected after customer update",
				zap.String("customer_id", req.CustomerID),
				zap.String("tx_ref", req.TransactionReference),
			)

			return &ProcessPaymentResponse{
				Success:            true,
				Message:            "payment processed successfully",
				CustomerID:         customer.ID,
				OutstandingBalance: customer.OutstandingBalance,
				TotalPaid:          customer.TotalPaid,
				PaymentProgress:    customer.GetPaymentProgress(),
				IsFullyPaid:        customer.IsFullyPaid(),
			}, nil
		}

		s.logger.Error("failed to save payment",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID),
		)
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	payment.MarkAsProcessed()

	s.logger.Info("payment processed successfully",
		zap.String("customer_id", req.CustomerID),
		zap.Int64("amount", req.TransactionAmount),
		zap.String("tx_ref", req.TransactionReference),
		zap.Int64("new_balance", customer.OutstandingBalance),
	)

	// Publish event asynchronously if event publisher is configured
	if s.eventPublisher != nil {
		go s.publishPaymentProcessedEvent(customer, req)
	}

	return &ProcessPaymentResponse{
		Success:            true,
		Message:            "payment processed successfully",
		CustomerID:         customer.ID,
		OutstandingBalance: customer.OutstandingBalance,
		TotalPaid:          customer.TotalPaid,
		PaymentProgress:    customer.GetPaymentProgress(),
		IsFullyPaid:        customer.IsFullyPaid(),
	}, nil
}

func (s *PaymentService) publishPaymentProcessedEvent(customer *domain.Customer, req ProcessPaymentRequest) {
	// Use background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := domain.NewPaymentProcessedEvent(customer.ID, domain.PaymentProcessedPayload{
		CustomerID:           customer.ID,
		TransactionReference: req.TransactionReference,
		Amount:               req.TransactionAmount,
		OutstandingBalance:   customer.OutstandingBalance,
		TotalPaid:            customer.TotalPaid,
		PaymentProgress:      customer.GetPaymentProgress(),
		IsFullyPaid:          customer.IsFullyPaid(),
		ProcessedAt:          time.Now(),
	})

	if err := s.eventPublisher.Publish(ctx, event); err != nil {
		s.logger.Error("failed to publish payment processed event",
			zap.Error(err),
			zap.String("customer_id", customer.ID),
			zap.String("event_id", event.GetEventID()),
		)
		// TODO: Save to outbox for retry
	} else {
		s.logger.Debug("payment processed event published",
			zap.String("event_id", event.GetEventID()),
			zap.String("customer_id", customer.ID),
		)
	}
}

func (s *PaymentService) GetCustomer(ctx context.Context, customerID string) (*domain.Customer, error) {
	return s.customerRepo.FindByID(ctx, customerID)
}

func (s *PaymentService) GetCustomerPayments(ctx context.Context, customerID string) ([]*domain.Payment, error) {
	// Verify customer exists
	_, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		s.logger.Error("failed to get customer",
			zap.Error(err),
			zap.String("customer_id", customerID),
		)
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	payments, err := s.paymentRepo.FindByCustomerID(ctx, customerID)
	if err != nil {
		s.logger.Error("failed to get customer payments",
			zap.Error(err),
			zap.String("customer_id", customerID),
		)
		return nil, fmt.Errorf("failed to get payments: %w", err)
	}

	s.logger.Info("retrieved customer payments",
		zap.String("customer_id", customerID),
		zap.Int("count", len(payments)),
	)

	return payments, nil
}
