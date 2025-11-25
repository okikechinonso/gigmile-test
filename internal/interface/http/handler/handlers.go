package handler

import (
	"github.com/gigmile/payment-service/internal/application/service"
	"github.com/gigmile/payment-service/internal/domain"
	sqlrepository "github.com/gigmile/payment-service/internal/infrastructure/repository/mysql"
	"go.uber.org/zap"
)

type Handlers struct {
	Payment *PaymentHandler
}

func NewHandlers(repos *sqlrepository.Repositories, eventPublisher domain.EventPublisher, logger *zap.Logger) *Handlers {
	paymentService := service.NewPaymentService(repos.Customer, repos.Payment, eventPublisher, logger)
	return &Handlers{
		Payment: NewPaymentHandler(paymentService, logger),
	}
}
