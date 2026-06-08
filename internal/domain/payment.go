package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Payment status constants
const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusApproved   = "APPROVED"
	StatusRejected   = "REJECTED"
	StatusFailed     = "FAILED"
)

// Domain-specific payment errors
var (
	ErrPaymentNotFound   = errors.New("payment not found")
	ErrInvalidAmount     = errors.New("payment amount must be greater than zero")
	ErrInvalidCurrency   = errors.New("invalid currency: only BRL, USD, and EUR are supported")
	ErrInvalidStatus     = errors.New("invalid payment status transition")
	ErrIdempotencyFailed = errors.New("idempotent validation failed")
)

// Payment represents the core transaction entity in the system.
type Payment struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Amount     float64
	Currency   string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Validate checks if the payment details are valid before saving.
func (p *Payment) Validate() error {
	if p.Amount <= 0 {
		return ErrInvalidAmount
	}
	if p.Currency != "BRL" && p.Currency != "USD" && p.Currency != "EUR" {
		return ErrInvalidCurrency
	}
	if p.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}
	return nil
}

// CanTransitionTo checks if a status transition is business-rule compliant.
func (p *Payment) CanTransitionTo(newStatus string) bool {
	// Business rules for transitions
	switch p.Status {
	case StatusPending:
		return newStatus == StatusProcessing || newStatus == StatusFailed
	case StatusProcessing:
		return newStatus == StatusApproved || newStatus == StatusRejected || newStatus == StatusFailed
	case StatusApproved, StatusRejected, StatusFailed:
		// Terminal states cannot transition further
		return false
	default:
		return false
	}
}

// TransitionTo changes the payment state if allowed.
func (p *Payment) TransitionTo(newStatus string) error {
	if !p.CanTransitionTo(newStatus) {
		return ErrInvalidStatus
	}
	p.Status = newStatus
	p.UpdatedAt = time.Now()
	return nil
}

// PaymentRepository defines the contract for persisting and retrieving payments.
type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	Update(ctx context.Context, payment *Payment) error
	List(ctx context.Context) ([]*Payment, error)
}
