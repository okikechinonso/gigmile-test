package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gigmile/payment-service/internal/application/service"
	"github.com/gigmile/payment-service/internal/interface/http/dto"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
	logger         *zap.Logger
}

func NewPaymentHandler(paymentService *service.PaymentService, logger *zap.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		logger:         logger,
	}
}

// ProcessPayment handles incoming payment webhook
func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	var req dto.PaymentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, "validation failed", err)
		return
	}

	// Parse amount and date
	amount, err := req.GetAmountInKobo()
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid transaction amount", err)
		return
	}

	txDate, err := req.GetTransactionDate()
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid transaction date", err)
		return
	}

	// Process payment
	result, err := h.paymentService.ProcessPayment(r.Context(), service.ProcessPaymentRequest{
		CustomerID:           req.CustomerID,
		PaymentStatus:        req.PaymentStatus,
		TransactionAmount:    amount,
		TransactionDate:      txDate,
		TransactionReference: req.TransactionReference,
	})

	if err != nil {
		h.logger.Error("failed to process payment",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID),
		)
		h.respondError(w, http.StatusInternalServerError, "failed to process payment", err)
		return
	}

	// Build response
	response := dto.PaymentResponse{
		Success:            result.Success,
		Message:            result.Message,
		CustomerID:         result.CustomerID,
		OutstandingBalance: result.OutstandingBalance,
		TotalPaid:          result.TotalPaid,
		PaymentProgress:    result.PaymentProgress,
		IsFullyPaid:        result.IsFullyPaid,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetCustomer retrieves customer information
func (h *PaymentHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "customer_id")

	if customerID == "" {
		h.respondError(w, http.StatusBadRequest, "customer_id is required", nil)
		return
	}

	customer, err := h.paymentService.GetCustomer(r.Context(), customerID)
	if err != nil {
		h.logger.Error("failed to get customer",
			zap.Error(err),
			zap.String("customer_id", customerID),
		)
		h.respondError(w, http.StatusNotFound, "customer not found", err)
		return
	}

	response := dto.CustomerResponse{
		CustomerID:         customer.ID,
		AssetValue:         customer.AssetValue,
		RepaymentTermWeeks: customer.RepaymentTermWeeks,
		OutstandingBalance: customer.OutstandingBalance,
		TotalPaid:          customer.TotalPaid,
		PaymentProgress:    customer.GetPaymentProgress(),
		Status:             string(customer.Status),
		IsFullyPaid:        customer.IsFullyPaid(),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HealthCheck handles health check endpoint
func (h *PaymentHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func (h *PaymentHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *PaymentHandler) respondError(w http.ResponseWriter, status int, message string, err error) {
	response := dto.ErrorResponse{
		Error:   message,
		Message: "",
	}

	if err != nil {
		response.Message = err.Error()
	}

	h.respondJSON(w, status, response)
}
