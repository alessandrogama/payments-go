package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/internal/domain/mocks"
	"github.com/google/uuid"
)

func TestOutboxProcessor_ProcessEvents(t *testing.T) {
	ctx := context.Background()
	interval := 10 * time.Millisecond
	limit := 5

	t.Run("successful processing", func(t *testing.T) {
		mockOutbox := new(mocks.MockOutboxRepository)
		mockPub := new(mocks.MockEventPublisher)
		processor := NewOutboxProcessor(mockOutbox, mockPub, interval, limit)

		eventID := uuid.New()
		event := &domain.OutboxEvent{
			ID:        eventID,
			EventType: "payments.created",
			Payload:   []byte(`{"id":"payment-id"}`),
			Status:    "PENDING",
		}

		// Mock Outbox Pending query
		mockOutbox.On("GetPending", ctx, limit).Return([]*domain.OutboxEvent{event}, nil)

		// Mock Publisher
		mockPub.On("Publish", ctx, "payments.created", eventID.String(), event.Payload).Return(nil)

		// Mock Outbox MarkProcessed
		mockOutbox.On("MarkProcessed", ctx, eventID).Return(nil)

		processor.processEvents(ctx)

		mockOutbox.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})

	t.Run("publisher failure marks event as failed", func(t *testing.T) {
		mockOutbox := new(mocks.MockOutboxRepository)
		mockPub := new(mocks.MockEventPublisher)
		processor := NewOutboxProcessor(mockOutbox, mockPub, interval, limit)

		eventID := uuid.New()
		event := &domain.OutboxEvent{
			ID:        eventID,
			EventType: "payments.created",
			Payload:   []byte(`{"id":"payment-id"}`),
			Status:    "PENDING",
			Attempts:  1,
		}

		mockOutbox.On("GetPending", ctx, limit).Return([]*domain.OutboxEvent{event}, nil)

		// Mock Publisher failure
		mockPub.On("Publish", ctx, "payments.created", eventID.String(), event.Payload).Return(errors.New("kafka connection reset"))

		// Mock Outbox MarkFailed with updated attempt count
		mockOutbox.On("MarkFailed", ctx, eventID, 2).Return(nil)

		processor.processEvents(ctx)

		mockOutbox.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})

	t.Run("no pending events does nothing", func(t *testing.T) {
		mockOutbox := new(mocks.MockOutboxRepository)
		mockPub := new(mocks.MockEventPublisher)
		processor := NewOutboxProcessor(mockOutbox, mockPub, interval, limit)

		mockOutbox.On("GetPending", ctx, limit).Return([]*domain.OutboxEvent{}, nil)

		processor.processEvents(ctx)

		mockOutbox.AssertExpectations(t)
		mockPub.AssertExpectations(t)
	})

	t.Run("outbox query error logs and returns", func(t *testing.T) {
		mockOutbox := new(mocks.MockOutboxRepository)
		mockPub := new(mocks.MockEventPublisher)
		processor := NewOutboxProcessor(mockOutbox, mockPub, interval, limit)

		mockOutbox.On("GetPending", ctx, limit).Return(([]*domain.OutboxEvent)(nil), errors.New("db error"))

		// Should not raise panics
		processor.processEvents(ctx)

		mockOutbox.AssertExpectations(t)
	})
}
