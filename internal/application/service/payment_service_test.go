package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gigmile/payment-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockCustomerRepository is a mock implementation of CustomerRepository
type MockCustomerRepository struct {
	mock.Mock
}

func (m *MockCustomerRepository) FindByID(ctx context.Context, customerID string) (*domain.Customer, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

func (m *MockCustomerRepository) Save(ctx context.Context, customer *domain.Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockCustomerRepository) UpdateBalance(ctx context.Context, customerID string, amount int64, version int64) error {
	args := m.Called(ctx, customerID, amount, version)
	return args.Error(0)
}

// MockPaymentRepository is a mock implementation of PaymentRepository
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) FindByTransactionReference(ctx context.Context, txRef string) (*domain.Payment, error) {
	args := m.Called(ctx, txRef)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentRepository) ExistsByTransactionReference(ctx context.Context, txRef string) (bool, error) {
	args := m.Called(ctx, txRef)
	return args.Bool(0), args.Error(1)
}

func (m *MockPaymentRepository) FindByCustomerID(ctx context.Context, customerID string) ([]*domain.Payment, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Payment), args.Error(1)
}

func (m *MockPaymentRepository) CountByCustomerID(ctx context.Context, customerID string) (int64, error) {
	args := m.Called(ctx, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPaymentRepository) FindByCustomerIDWithPagination(ctx context.Context, customerID string, limit, offset int) ([]*domain.Payment, error) {
	args := m.Called(ctx, customerID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Payment), args.Error(1)
}

func TestGetCustomerPayments_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	customerID := "GIG00001"
	logger := zap.NewNop()

	mockCustomerRepo := new(MockCustomerRepository)
	mockPaymentRepo := new(MockPaymentRepository)

	service := NewPaymentService(mockCustomerRepo, mockPaymentRepo, nil, logger)

	// Mock customer exists
	customer := &domain.Customer{
		ID:                 customerID,
		AssetValue:         100000000,
		OutstandingBalance: 50000000,
		TotalPaid:          50000000,
		Status:             domain.CustomerStatusActive,
		Version:            1,
	}
	mockCustomerRepo.On("FindByID", ctx, customerID).Return(customer, nil)

	// Mock payments
	now := time.Now()
	payments := []*domain.Payment{
		{
			ID:                   "payment-1",
			CustomerID:           customerID,
			Amount:               25000000,
			TransactionReference: "TXN001",
			TransactionDate:      now.Add(-48 * time.Hour),
			Status:               domain.PaymentStatusComplete,
			ProcessedAt:          now.Add(-48 * time.Hour),
		},
		{
			ID:                   "payment-2",
			CustomerID:           customerID,
			Amount:               25000000,
			TransactionReference: "TXN002",
			TransactionDate:      now.Add(-24 * time.Hour),
			Status:               domain.PaymentStatusComplete,
			ProcessedAt:          now.Add(-24 * time.Hour),
		},
	}
	mockPaymentRepo.On("FindByCustomerID", ctx, customerID).Return(payments, nil)

	// Act
	result, err := service.GetCustomerPayments(ctx, customerID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, "payment-1", result[0].ID)
	assert.Equal(t, "payment-2", result[1].ID)
	assert.Equal(t, int64(25000000), result[0].Amount)
	assert.Equal(t, "TXN001", result[0].TransactionReference)

	mockCustomerRepo.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
}

func TestGetCustomerPayments_CustomerNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	customerID := "NONEXISTENT"
	logger := zap.NewNop()

	mockCustomerRepo := new(MockCustomerRepository)
	mockPaymentRepo := new(MockPaymentRepository)

	service := NewPaymentService(mockCustomerRepo, mockPaymentRepo, nil, logger)

	// Mock customer not found
	mockCustomerRepo.On("FindByID", ctx, customerID).Return(nil, errors.New("customer not found"))

	// Act
	result, err := service.GetCustomerPayments(ctx, customerID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get customer")

	mockCustomerRepo.AssertExpectations(t)
	mockPaymentRepo.AssertNotCalled(t, "FindByCustomerID")
}

func TestGetCustomerPayments_NoPayments(t *testing.T) {
	// Arrange
	ctx := context.Background()
	customerID := "GIG00002"
	logger := zap.NewNop()

	mockCustomerRepo := new(MockCustomerRepository)
	mockPaymentRepo := new(MockPaymentRepository)

	service := NewPaymentService(mockCustomerRepo, mockPaymentRepo, nil, logger)

	// Mock customer exists
	customer := &domain.Customer{
		ID:                 customerID,
		AssetValue:         100000000,
		OutstandingBalance: 100000000,
		TotalPaid:          0,
		Status:             domain.CustomerStatusActive,
		Version:            1,
	}
	mockCustomerRepo.On("FindByID", ctx, customerID).Return(customer, nil)

	// Mock no payments
	mockPaymentRepo.On("FindByCustomerID", ctx, customerID).Return([]*domain.Payment{}, nil)

	// Act
	result, err := service.GetCustomerPayments(ctx, customerID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)

	mockCustomerRepo.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
}

func TestGetCustomerPayments_PaymentRepoError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	customerID := "GIG00003"
	logger := zap.NewNop()

	mockCustomerRepo := new(MockCustomerRepository)
	mockPaymentRepo := new(MockPaymentRepository)

	service := NewPaymentService(mockCustomerRepo, mockPaymentRepo, nil, logger)

	// Mock customer exists
	customer := &domain.Customer{
		ID:                 customerID,
		AssetValue:         100000000,
		OutstandingBalance: 50000000,
		TotalPaid:          50000000,
		Status:             domain.CustomerStatusActive,
		Version:            1,
	}
	mockCustomerRepo.On("FindByID", ctx, customerID).Return(customer, nil)

	// Mock payment repository error
	mockPaymentRepo.On("FindByCustomerID", ctx, customerID).Return(nil, errors.New("database connection error"))

	// Act
	result, err := service.GetCustomerPayments(ctx, customerID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get payments")

	mockCustomerRepo.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
}

func TestGetCustomerPayments_MultiplePayments(t *testing.T) {
	// Arrange
	ctx := context.Background()
	customerID := "GIG00004"
	logger := zap.NewNop()

	mockCustomerRepo := new(MockCustomerRepository)
	mockPaymentRepo := new(MockPaymentRepository)

	service := NewPaymentService(mockCustomerRepo, mockPaymentRepo, nil, logger)

	// Mock customer exists
	customer := &domain.Customer{
		ID:                 customerID,
		AssetValue:         100000000,
		OutstandingBalance: 0,
		TotalPaid:          100000000,
		Status:             domain.CustomerStatusCompleted,
		Version:            10,
	}
	mockCustomerRepo.On("FindByID", ctx, customerID).Return(customer, nil)

	// Mock multiple payments (customer made 10 payments)
	now := time.Now()
	payments := make([]*domain.Payment, 10)
	for i := 0; i < 10; i++ {
		payments[i] = &domain.Payment{
			ID:                   "payment-" + string(rune(i)),
			CustomerID:           customerID,
			Amount:               10000000,
			TransactionReference: "TXN00" + string(rune(i)),
			TransactionDate:      now.Add(-time.Duration(i*24) * time.Hour),
			Status:               domain.PaymentStatusComplete,
			ProcessedAt:          now.Add(-time.Duration(i*24) * time.Hour),
		}
	}
	mockPaymentRepo.On("FindByCustomerID", ctx, customerID).Return(payments, nil)

	// Act
	result, err := service.GetCustomerPayments(ctx, customerID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 10)

	mockCustomerRepo.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
}

func TestGetCustomerPayments_ContextCancellation(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	customerID := "GIG00005"
	logger := zap.NewNop()

	mockCustomerRepo := new(MockCustomerRepository)
	mockPaymentRepo := new(MockPaymentRepository)

	service := NewPaymentService(mockCustomerRepo, mockPaymentRepo, nil, logger)

	// Mock customer repo returns context error
	mockCustomerRepo.On("FindByID", ctx, customerID).Return(nil, context.Canceled)

	// Act
	result, err := service.GetCustomerPayments(ctx, customerID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	mockCustomerRepo.AssertExpectations(t)
}
