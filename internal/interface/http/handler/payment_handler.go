package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

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

// GetCustomerPayments retrieves all payments for a customer
func (h *PaymentHandler) GetCustomerPayments(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		h.respondError(w, http.StatusBadRequest, "customer_id is required", nil)
		return
	}

	// Parse pagination parameters
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	// If pagination params are provided, use paginated endpoint
	if pageStr != "" || pageSizeStr != "" {
		h.getCustomerPaymentsPaginated(w, r, customerID, pageStr, pageSizeStr)
		return
	}

	// Otherwise, return all payments (backward compatibility)
	payments, err := h.paymentService.GetCustomerPayments(r.Context(), customerID)
	if err != nil {
		h.logger.Error("failed to get customer payments",
			zap.Error(err),
			zap.String("customer_id", customerID),
		)
		h.respondError(w, http.StatusInternalServerError, "failed to get customer payments", err)
		return
	}

	// Transform domain payments to response DTOs
	response := make([]dto.PaymentRecordResponse, len(payments))
	for i, payment := range payments {
		response[i] = dto.PaymentRecordResponse{
			ID:                   payment.ID,
			CustomerID:           payment.CustomerID,
			TransactionAmount:    payment.Amount,
			TransactionReference: payment.TransactionReference,
			TransactionDate:      payment.TransactionDate.Format("2006-01-02T15:04:05Z07:00"),
			Status:               string(payment.Status),
			ProcessedAt:          payment.ProcessedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	h.logger.Info("customer payments retrieved successfully",
		zap.String("customer_id", customerID),
		zap.Int("count", len(payments)),
	)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"customer_id": customerID,
		"count":       len(payments),
		"payments":    response,
	})
}

func (h *PaymentHandler) getCustomerPaymentsPaginated(w http.ResponseWriter, r *http.Request, customerID, pageStr, pageSizeStr string) {
	page := 1
	pageSize := 10

	// Parse page
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Parse page_size
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	params := service.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.paymentService.GetCustomerPaymentsPaginated(r.Context(), customerID, params)
	if err != nil {
		h.logger.Error("failed to get customer payments with pagination",
			zap.Error(err),
			zap.String("customer_id", customerID),
			zap.Int("page", page),
			zap.Int("page_size", pageSize),
		)
		h.respondError(w, http.StatusInternalServerError, "failed to get customer payments", err)
		return
	}

	// Transform domain payments to response DTOs
	response := make([]dto.PaymentRecordResponse, len(result.Payments))
	for i, payment := range result.Payments {
		response[i] = dto.PaymentRecordResponse{
			ID:                   payment.ID,
			CustomerID:           payment.CustomerID,
			TransactionAmount:    payment.Amount,
			TransactionReference: payment.TransactionReference,
			TransactionDate:      payment.TransactionDate.Format("2006-01-02T15:04:05Z07:00"),
			Status:               string(payment.Status),
			ProcessedAt:          payment.ProcessedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	h.logger.Info("customer payments retrieved successfully with pagination",
		zap.String("customer_id", customerID),
		zap.Int("count", len(response)),
		zap.Int("page", result.Page),
		zap.Int("page_size", result.PageSize),
		zap.Int64("total_count", result.TotalCount),
	)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"customer_id": customerID,
		"payments":    response,
		"pagination": map[string]interface{}{
			"page":        result.Page,
			"page_size":   result.PageSize,
			"total_count": result.TotalCount,
			"total_pages": result.TotalPages,
		},
	})
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
