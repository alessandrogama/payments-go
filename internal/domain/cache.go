package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PaymentCache defines the contract for caching payment queries to reduce DB load.
type PaymentCache interface {
	Get(ctx context.Context, id uuid.UUID) (*Payment, error)
	Set(ctx context.Context, payment *Payment, ttl time.Duration) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// IdempotencyManager guarantees that multiple submissions with the same key process only once.
type IdempotencyManager interface {
	// TryAcquire checks if an idempotency key exists.
	// If it exists, returns the associated payment ID and true.
	// If not, registers the key mapped to the payment ID and returns the payment ID and false.
	TryAcquire(ctx context.Context, key string, paymentID uuid.UUID, ttl time.Duration) (uuid.UUID, bool, error)
}
