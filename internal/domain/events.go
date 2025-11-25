package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	EventTypePaymentReceived  = "payment.received"
	EventTypePaymentProcessed = "payment.processed"
	EventTypePaymentFailed    = "payment.failed"
	EventTypeCustomerUpdated  = "customer.updated"
)

// DomainEvent represents a domain event
type DomainEvent interface {
	GetEventID() string
	GetEventType() string
	GetAggregateID() string
	GetOccurredAt() time.Time
	GetPayload() interface{}
}

// BaseEvent provides common event fields
type BaseEvent struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	AggregateID string    `json:"aggregate_id"`
	OccurredAt  time.Time `json:"occurred_at"`
}

func (e BaseEvent) GetEventID() string       { return e.EventID }
func (e BaseEvent) GetEventType() string     { return e.EventType }
func (e BaseEvent) GetAggregateID() string   { return e.AggregateID }
func (e BaseEvent) GetOccurredAt() time.Time { return e.OccurredAt }

// PaymentProcessedEvent - Payment successfully applied
type PaymentProcessedEvent struct {
	BaseEvent
	Payload PaymentProcessedPayload `json:"payload"`
}

func (e PaymentProcessedEvent) GetPayload() interface{} { return e.Payload }

type PaymentProcessedPayload struct {
	CustomerID           string    `json:"customer_id"`
	TransactionReference string    `json:"transaction_reference"`
	Amount               int64     `json:"amount"`
	OutstandingBalance   int64     `json:"outstanding_balance"`
	TotalPaid            int64     `json:"total_paid"`
	PaymentProgress      float64   `json:"payment_progress"`
	IsFullyPaid          bool      `json:"is_fully_paid"`
	ProcessedAt          time.Time `json:"processed_at"`
}

func NewPaymentProcessedEvent(customerID string, payload PaymentProcessedPayload) *PaymentProcessedEvent {
	return &PaymentProcessedEvent{
		BaseEvent: BaseEvent{
			EventID:     uuid.New().String(),
			EventType:   EventTypePaymentProcessed,
			AggregateID: customerID,
			OccurredAt:  time.Now(),
		},
		Payload: payload,
	}
}

// EventPublisher interface
type EventPublisher interface {
	Publish(ctx context.Context, event DomainEvent) error
}

// EventSubscriber interface
type EventSubscriber interface {
	Subscribe(ctx context.Context, eventType string, handler EventHandler) error
}

// EventHandler processes events
type EventHandler func(ctx context.Context, event DomainEvent) error
