package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aless/gopay-processing-engine/internal/application"
	"github.com/aless/gopay-processing-engine/internal/domain"
	internalHttp "github.com/aless/gopay-processing-engine/internal/interfaces/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPaymentService is a mock implementation of application.PaymentService
type MockPaymentService struct {
	mock.Mock
}

func (m *MockPaymentService) CreatePayment(ctx context.Context, input application.CreatePaymentInput) (*domain.Payment, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentService) GetPaymentByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentService) ListPayments(ctx context.Context) ([]*domain.Payment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Payment), args.Error(1)
}

func TestPaymentHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	customerID := uuid.New()

	t.Run("successful creation", func(t *testing.T) {
		mockService := new(MockPaymentService)
		handler := internalHttp.NewPaymentHandler(mockService)

		r := gin.Default()
		r.POST("/payments", handler.Create)

		input := application.CreatePaymentInput{
			CustomerID:     customerID,
			Amount:         150.0,
			Currency:       "USD",
			IdempotencyKey: "unique-key-1",
		}

		payment := &domain.Payment{
			ID:         uuid.New(),
			CustomerID: customerID,
			Amount:     150.0,
			Currency:   "USD",
			Status:     domain.StatusPending,
		}

		mockService.On("CreatePayment", mock.Anything, input).Return(payment, nil)

		body := map[string]interface{}{
			"customer_id": customerID.String(),
			"amount":      150.0,
			"currency":    "USD",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/payments", bytes.NewBuffer(jsonBody))
		req.Header.Set("Idempotency-Key", "unique-key-1")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), payment.ID.String())
		mockService.AssertExpectations(t)
	})

	t.Run("missing idempotency key", func(t *testing.T) {
		mockService := new(MockPaymentService)
		handler := internalHttp.NewPaymentHandler(mockService)

		r := gin.Default()
		r.POST("/payments", handler.Create)

		body := map[string]interface{}{
			"customer_id": customerID.String(),
			"amount":      150.0,
			"currency":    "USD",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/payments", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Idempotency-Key header is required")
		mockService.AssertNotCalled(t, "CreatePayment")
	})

	t.Run("invalid customer UUID", func(t *testing.T) {
		mockService := new(MockPaymentService)
		handler := internalHttp.NewPaymentHandler(mockService)

		r := gin.Default()
		r.POST("/payments", handler.Create)

		body := map[string]interface{}{
			"customer_id": "invalid-uuid",
			"amount":      150.0,
			"currency":    "USD",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/payments", bytes.NewBuffer(jsonBody))
		req.Header.Set("Idempotency-Key", "unique-key-1")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "expected UUID")
	})

	t.Run("idempotency conflict error", func(t *testing.T) {
		mockService := new(MockPaymentService)
		handler := internalHttp.NewPaymentHandler(mockService)

		r := gin.Default()
		r.POST("/payments", handler.Create)

		mockService.On("CreatePayment", mock.Anything, mock.Anything).
			Return((*domain.Payment)(nil), domain.ErrIdempotencyFailed)

		body := map[string]interface{}{
			"customer_id": customerID.String(),
			"amount":      150.0,
			"currency":    "USD",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/payments", bytes.NewBuffer(jsonBody))
		req.Header.Set("Idempotency-Key", "conflict-key")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "idempotency conflict")
	})
}

func TestPaymentHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paymentID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		mockService := new(MockPaymentService)
		handler := internalHttp.NewPaymentHandler(mockService)

		r := gin.Default()
		r.GET("/payments/:id", handler.GetByID)

		payment := &domain.Payment{
			ID:     paymentID,
			Amount: 100.0,
		}
		mockService.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)

		req, _ := http.NewRequest("GET", "/payments/"+paymentID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), paymentID.String())
		mockService.AssertExpectations(t)
	})

	t.Run("payment not found error", func(t *testing.T) {
		mockService := new(MockPaymentService)
		handler := internalHttp.NewPaymentHandler(mockService)

		r := gin.Default()
		r.GET("/payments/:id", handler.GetByID)

		mockService.On("GetPaymentByID", mock.Anything, paymentID).
			Return((*domain.Payment)(nil), domain.ErrPaymentNotFound)

		req, _ := http.NewRequest("GET", "/payments/"+paymentID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestPaymentHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful list", func(t *testing.T) {
		mockService := new(MockPaymentService)
		handler := internalHttp.NewPaymentHandler(mockService)

		r := gin.Default()
		r.GET("/payments", handler.List)

		payments := []*domain.Payment{
			{ID: uuid.New(), Amount: 10},
			{ID: uuid.New(), Amount: 20},
		}
		mockService.On("ListPayments", mock.Anything).Return(payments, nil)

		req, _ := http.NewRequest("GET", "/payments", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), payments[0].ID.String())
		mockService.AssertExpectations(t)
	})
}
