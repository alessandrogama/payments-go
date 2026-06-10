package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aless/gopay-processing-engine/internal/application"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPaymentService_CreatePayment(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	idempotencyKey := "key-1234"
	amount := 250.0
	currency := "USD"
	redisTTL := 1 * time.Hour

	input := application.CreatePaymentInput{
		CustomerID:     customerID,
		Amount:         amount,
		Currency:       currency,
		IdempotencyKey: idempotencyKey,
	}

	t.Run("successful payment creation", func(t *testing.T) {
		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockIdemp := new(mocks.MockIdempotencyManager)
		mockOutbox := new(mocks.MockOutboxRepository)

		service := application.NewPaymentService(mockPayRepo, mockCache, mockIdemp, mockOutbox, db, redisTTL)

		// 1. Mock Idempotency check: key is not duplicate
		mockIdemp.On("TryAcquire", ctx, idempotencyKey, mock.Anything, 24*time.Hour).
			Return(uuid.Nil, false, nil)

		// 2. Mock Transaction expectation
		sqlMock.ExpectBegin()
		sqlMock.ExpectCommit()

		// 3. Mock Repositories
		mockPayRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
		mockOutbox.On("Save", mock.Anything, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil)

		// 4. Mock Cache populate
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.Payment"), redisTTL).Return(nil)

		payment, err := service.CreatePayment(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, domain.StatusPending, payment.Status)
		assert.Equal(t, amount, payment.Amount)
		assert.Equal(t, currency, payment.Currency)

		assert.NoError(t, sqlMock.ExpectationsWereMet())
		mockPayRepo.AssertExpectations(t)
		mockOutbox.AssertExpectations(t)
		mockCache.AssertExpectations(t)
		mockIdemp.AssertExpectations(t)
	})

	t.Run("duplicate idempotency key - cache hit", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockIdemp := new(mocks.MockIdempotencyManager)
		mockOutbox := new(mocks.MockOutboxRepository)

		service := application.NewPaymentService(mockPayRepo, mockCache, mockIdemp, mockOutbox, db, redisTTL)

		existingPaymentID := uuid.New()
		existingPayment := &domain.Payment{
			ID:         existingPaymentID,
			CustomerID: customerID,
			Amount:     amount,
			Currency:   currency,
			Status:     domain.StatusApproved,
		}

		// Mock Idempotency check: key is duplicate, maps to existingPaymentID
		mockIdemp.On("TryAcquire", ctx, idempotencyKey, mock.Anything, 24*time.Hour).
			Return(existingPaymentID, true, nil)

		// Mock Cache hit
		mockCache.On("Get", ctx, existingPaymentID).Return(existingPayment, nil)

		payment, err := service.CreatePayment(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, existingPaymentID, payment.ID)
		assert.Equal(t, domain.StatusApproved, payment.Status)

		mockIdemp.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("duplicate idempotency key - cache miss db hit", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockIdemp := new(mocks.MockIdempotencyManager)
		mockOutbox := new(mocks.MockOutboxRepository)

		service := application.NewPaymentService(mockPayRepo, mockCache, mockIdemp, mockOutbox, db, redisTTL)

		existingPaymentID := uuid.New()
		existingPayment := &domain.Payment{
			ID:         existingPaymentID,
			CustomerID: customerID,
			Amount:     amount,
			Currency:   currency,
			Status:     domain.StatusApproved,
		}

		mockIdemp.On("TryAcquire", ctx, idempotencyKey, mock.Anything, 24*time.Hour).
			Return(existingPaymentID, true, nil)

		// Cache miss
		mockCache.On("Get", ctx, existingPaymentID).Return((*domain.Payment)(nil), errors.New("cache miss"))
		// DB hit
		mockPayRepo.On("GetByID", ctx, existingPaymentID).Return(existingPayment, nil)
		// Cache set updated
		mockCache.On("Set", ctx, existingPayment, redisTTL).Return(nil)

		payment, err := service.CreatePayment(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, existingPaymentID, payment.ID)

		mockIdemp.AssertExpectations(t)
		mockCache.AssertExpectations(t)
		mockPayRepo.AssertExpectations(t)
	})

	t.Run("missing idempotency key error", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		service := application.NewPaymentService(nil, nil, nil, nil, db, redisTTL)

		badInput := input
		badInput.IdempotencyKey = ""

		payment, err := service.CreatePayment(ctx, badInput)
		assert.Error(t, err)
		assert.Nil(t, payment)
		assert.Contains(t, err.Error(), "idempotency key is required")
	})

	t.Run("db transaction rollback on save error", func(t *testing.T) {
		db, sqlMock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockIdemp := new(mocks.MockIdempotencyManager)
		mockOutbox := new(mocks.MockOutboxRepository)

		service := application.NewPaymentService(mockPayRepo, mockCache, mockIdemp, mockOutbox, db, redisTTL)

		mockIdemp.On("TryAcquire", ctx, idempotencyKey, mock.Anything, 24*time.Hour).
			Return(uuid.Nil, false, nil)

		sqlMock.ExpectBegin()
		sqlMock.ExpectRollback()

		// Return error on Create
		mockPayRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

		payment, err := service.CreatePayment(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, payment)

		assert.NoError(t, sqlMock.ExpectationsWereMet())
		mockPayRepo.AssertExpectations(t)
		mockIdemp.AssertExpectations(t)
	})
}

func TestPaymentService_GetPaymentByID(t *testing.T) {
	ctx := context.Background()
	paymentID := uuid.New()
	redisTTL := 1 * time.Hour

	payment := &domain.Payment{
		ID:         paymentID,
		CustomerID: uuid.New(),
		Amount:     120.0,
		Currency:   "BRL",
		Status:     domain.StatusPending,
	}

	t.Run("cache hit", func(t *testing.T) {
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		service := application.NewPaymentService(mockPayRepo, mockCache, nil, nil, nil, redisTTL)

		mockCache.On("Get", ctx, paymentID).Return(payment, nil)

		result, err := service.GetPaymentByID(ctx, paymentID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, paymentID, result.ID)

		mockCache.AssertExpectations(t)
	})

	t.Run("cache miss - db hit", func(t *testing.T) {
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		service := application.NewPaymentService(mockPayRepo, mockCache, nil, nil, nil, redisTTL)

		mockCache.On("Get", ctx, paymentID).Return((*domain.Payment)(nil), errors.New("cache miss"))
		mockPayRepo.On("GetByID", ctx, paymentID).Return(payment, nil)
		mockCache.On("Set", ctx, payment, redisTTL).Return(nil)

		result, err := service.GetPaymentByID(ctx, paymentID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, paymentID, result.ID)

		mockCache.AssertExpectations(t)
		mockPayRepo.AssertExpectations(t)
	})
}

func TestPaymentService_ListPayments(t *testing.T) {
	ctx := context.Background()
	mockPayRepo := new(mocks.MockPaymentRepository)
	service := application.NewPaymentService(mockPayRepo, nil, nil, nil, nil, 1*time.Hour)

	payments := []*domain.Payment{
		{ID: uuid.New(), Amount: 100},
		{ID: uuid.New(), Amount: 200},
	}

	mockPayRepo.On("List", ctx).Return(payments, nil)

	result, err := service.ListPayments(ctx)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, payments[0].ID, result[0].ID)

	mockPayRepo.AssertExpectations(t)
}
