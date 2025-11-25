package router

import (
	"time"

	"github.com/gigmile/payment-service/internal/interface/http/handler"
	"github.com/gigmile/payment-service/internal/interface/http/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func NewRouter(handlers *handler.Handlers, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logger(logger))
	r.Use(chimiddleware.Compress(5))
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Routes
	r.Get("/health", handlers.Payment.HealthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/payments", handlers.Payment.ProcessPayment)
		r.Get("/payments", handlers.Payment.GetCustomerPayments)
		r.Get("/customers/{customer_id}", handlers.Payment.GetCustomer)
	})

	return r
}
