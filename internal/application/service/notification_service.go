package service

import (
	"context"
	"fmt"

	"github.com/gigmile/payment-service/internal/domain"
	"go.uber.org/zap"
)

// NotificationService handles side effects like SMS, emails, etc.
type NotificationService struct {
	customerRepo domain.CustomerRepository
	logger       *zap.Logger
}

func NewNotificationService(
	customerRepo domain.CustomerRepository,
	logger *zap.Logger,
) *NotificationService {
	return &NotificationService{
		customerRepo: customerRepo,
		logger:       logger,
	}
}

// HandlePaymentProcessed handles payment processed events
func (s *NotificationService) HandlePaymentProcessed(ctx context.Context, event domain.DomainEvent) error {
	paymentEvent, ok := event.(*domain.PaymentProcessedEvent)
	if !ok {
		return fmt.Errorf("invalid event type")
	}

	payload := paymentEvent.Payload

	s.logger.Info("handling payment processed event",
		zap.String("event_id", event.GetEventID()),
		zap.String("customer_id", payload.CustomerID),
		zap.Int64("amount", payload.Amount),
	)

	// TODO: Implement actual notification logic
	// Examples:
	// - Send SMS: "Payment of N%d received. Balance: N%d"
	// - Send Email receipt
	// - Update analytics dashboard
	// - Trigger loyalty points
	// - Generate invoice

	// Simulate SMS sending
	s.logger.Info("SMS notification sent",
		zap.String("customer_id", payload.CustomerID),
		zap.String("message", fmt.Sprintf("Payment of N%d received. Outstanding balance: N%d",
			payload.Amount/100, payload.OutstandingBalance/100)),
	)

	// If customer fully paid, send congratulations
	if payload.IsFullyPaid {
		s.logger.Info("Congratulations SMS sent",
			zap.String("customer_id", payload.CustomerID),
			zap.String("message", "Congratulations! You now own your asset!"),
		)
	}

	return nil
}
