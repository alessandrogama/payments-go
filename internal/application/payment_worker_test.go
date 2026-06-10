package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPaymentWorker_ProcessEvent(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		KafkaTopicCreated:   "payments.created",
		KafkaTopicProcessed: "payments.processed",
		KafkaTopicFailed:    "payments.failed",
		KafkaTopicDLQ:       "payments.dlq",
		RedisTTL:            1 * time.Hour,
	}

	t.Run("corrupt payload routed to DLQ", func(t *testing.T) {
		mockConsumer := new(mocks.MockKafkaConsumer)
		mockPub := new(mocks.MockEventPublisher)
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockGateway := new(mocks.MockPaymentGateway)

		worker := NewPaymentWorker(cfg, mockConsumer, mockPub, mockPayRepo, mockCache, mockGateway)

		corruptPayload := []byte(`{"id": "invalid-uuid-format"}`)

		// Expect routing to DLQ
		mockPub.On("Publish", ctx, cfg.KafkaTopicDLQ, mock.Anything, mock.Anything).Return(nil)

		err := worker.ProcessEvent(ctx, "test-key", corruptPayload)
		assert.NoError(t, err) // Should return nil so offset is committed

		mockPub.AssertExpectations(t)
	})

	t.Run("payment not found routed to DLQ", func(t *testing.T) {
		mockConsumer := new(mocks.MockKafkaConsumer)
		mockPub := new(mocks.MockEventPublisher)
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockGateway := new(mocks.MockPaymentGateway)

		worker := NewPaymentWorker(cfg, mockConsumer, mockPub, mockPayRepo, mockCache, mockGateway)

		paymentID := uuid.New()
		paymentPayload, _ := json.Marshal(domain.Payment{ID: paymentID})

		// Mock repo GetByID returns ErrPaymentNotFound
		mockPayRepo.On("GetByID", ctx, paymentID).Return((*domain.Payment)(nil), domain.ErrPaymentNotFound)

		// Expect routing to DLQ
		mockPub.On("Publish", ctx, cfg.KafkaTopicDLQ, mock.Anything, mock.Anything).Return(nil)

		err := worker.ProcessEvent(ctx, "test-key", paymentPayload)
		assert.NoError(t, err)

		mockPayRepo.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})

	t.Run("payment is not pending skipped", func(t *testing.T) {
		mockConsumer := new(mocks.MockKafkaConsumer)
		mockPub := new(mocks.MockEventPublisher)
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockGateway := new(mocks.MockPaymentGateway)

		worker := NewPaymentWorker(cfg, mockConsumer, mockPub, mockPayRepo, mockCache, mockGateway)

		paymentID := uuid.New()
		paymentPayload, _ := json.Marshal(domain.Payment{ID: paymentID})

		// Return an already APPROVED payment
		existingPayment := &domain.Payment{
			ID:     paymentID,
			Status: domain.StatusApproved,
		}
		mockPayRepo.On("GetByID", ctx, paymentID).Return(existingPayment, nil)

		err := worker.ProcessEvent(ctx, "test-key", paymentPayload)
		assert.NoError(t, err)

		mockPayRepo.AssertExpectations(t)
		mockPub.AssertNotCalled(t, "Publish")
	})

	t.Run("successful approval flow", func(t *testing.T) {
		mockConsumer := new(mocks.MockKafkaConsumer)
		mockPub := new(mocks.MockEventPublisher)
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockGateway := new(mocks.MockPaymentGateway)

		worker := NewPaymentWorker(cfg, mockConsumer, mockPub, mockPayRepo, mockCache, mockGateway)

		paymentID := uuid.New()
		paymentPayload, _ := json.Marshal(domain.Payment{ID: paymentID})

		payment := &domain.Payment{
			ID:     paymentID,
			Status: domain.StatusPending,
		}

		mockPayRepo.On("GetByID", ctx, paymentID).Return(payment, nil)

		// State transition update in DB (PROCESSING status)
		mockPayRepo.On("Update", ctx, mock.AnythingOfType("*domain.Payment")).Return(nil).Once()
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.Payment"), cfg.RedisTTL).Return(nil).Once()

		// Gateway Process mock
		gatewayResp := &domain.GatewayResponse{
			Status:        domain.StatusApproved,
			TransactionID: "tx-12345",
		}
		mockGateway.On("Process", ctx, mock.AnythingOfType("*domain.Payment")).Return(gatewayResp, nil)

		// State transition update in DB (APPROVED status)
		mockPayRepo.On("Update", ctx, mock.AnythingOfType("*domain.Payment")).Return(nil).Once()
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.Payment"), cfg.RedisTTL).Return(nil).Once()

		// Outcome publish mock
		mockPub.On("Publish", ctx, cfg.KafkaTopicProcessed, paymentID.String(), mock.Anything).Return(nil)

		err := worker.ProcessEvent(ctx, "test-key", paymentPayload)
		assert.NoError(t, err)

		assert.Equal(t, domain.StatusApproved, payment.Status)

		mockPayRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
		mockGateway.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})

	t.Run("gateway reject flow", func(t *testing.T) {
		mockConsumer := new(mocks.MockKafkaConsumer)
		mockPub := new(mocks.MockEventPublisher)
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockGateway := new(mocks.MockPaymentGateway)

		worker := NewPaymentWorker(cfg, mockConsumer, mockPub, mockPayRepo, mockCache, mockGateway)

		paymentID := uuid.New()
		paymentPayload, _ := json.Marshal(domain.Payment{ID: paymentID})

		payment := &domain.Payment{
			ID:     paymentID,
			Status: domain.StatusPending,
		}

		mockPayRepo.On("GetByID", ctx, paymentID).Return(payment, nil)

		mockPayRepo.On("Update", ctx, mock.Anything).Return(nil).Once()
		mockCache.On("Set", ctx, mock.Anything, cfg.RedisTTL).Return(nil).Once()

		gatewayResp := &domain.GatewayResponse{
			Status:       domain.StatusRejected,
			ErrorMessage: "insufficient funds",
		}
		mockGateway.On("Process", ctx, mock.Anything).Return(gatewayResp, nil)

		mockPayRepo.On("Update", ctx, mock.Anything).Return(nil).Once()
		mockCache.On("Set", ctx, mock.Anything, cfg.RedisTTL).Return(nil).Once()

		mockPub.On("Publish", ctx, cfg.KafkaTopicFailed, paymentID.String(), mock.Anything).Return(nil)

		err := worker.ProcessEvent(ctx, "test-key", paymentPayload)
		assert.NoError(t, err)

		assert.Equal(t, domain.StatusRejected, payment.Status)

		mockPayRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
		mockGateway.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})

	t.Run("gateway failure retries and eventual DLQ routing", func(t *testing.T) {
		mockConsumer := new(mocks.MockKafkaConsumer)
		mockPub := new(mocks.MockEventPublisher)
		mockPayRepo := new(mocks.MockPaymentRepository)
		mockCache := new(mocks.MockPaymentCache)
		mockGateway := new(mocks.MockPaymentGateway)

		worker := NewPaymentWorker(cfg, mockConsumer, mockPub, mockPayRepo, mockCache, mockGateway)

		paymentID := uuid.New()
		paymentPayload, _ := json.Marshal(domain.Payment{ID: paymentID})

		payment := &domain.Payment{
			ID:     paymentID,
			Status: domain.StatusPending,
		}

		mockPayRepo.On("GetByID", ctx, paymentID).Return(payment, nil)

		mockPayRepo.On("Update", ctx, mock.Anything).Return(nil).Once()
		mockCache.On("Set", ctx, mock.Anything, cfg.RedisTTL).Return(nil).Once()

		// Gateway returns error 3 times
		mockGateway.On("Process", ctx, mock.Anything).Return((*domain.GatewayResponse)(nil), errors.New("timeout")).Times(3)

		// Expect payment updated to FAILED and published to failed topic
		mockPayRepo.On("Update", ctx, mock.Anything).Return(nil).Once()
		mockCache.On("Set", ctx, mock.Anything, cfg.RedisTTL).Return(nil).Once()
		mockPub.On("Publish", ctx, cfg.KafkaTopicFailed, paymentID.String(), mock.Anything).Return(nil)

		// Expect routing to DLQ
		mockPub.On("Publish", ctx, cfg.KafkaTopicDLQ, mock.Anything, mock.Anything).Return(nil)

		err := worker.ProcessEvent(ctx, "test-key", paymentPayload)
		assert.NoError(t, err)

		assert.Equal(t, domain.StatusFailed, payment.Status)

		mockPayRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
		mockGateway.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})
}
