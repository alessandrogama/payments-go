package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Outbox event status constants
const (
	OutboxStatusPending   = "PENDING"
	OutboxStatusProcessed = "PROCESSED"
	OutboxStatusFailed    = "FAILED"
)

// OutboxEvent represents an event stored in the DB to be processed asynchronously.
type OutboxEvent struct {
	ID          uuid.UUID
	EventType   string
	Payload     []byte
	Status      string
	Attempts    int
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

// OutboxRepository defines operations for the transactional outbox pattern.
type OutboxRepository interface {
	Save(ctx context.Context, event *OutboxEvent) error
	GetPending(ctx context.Context, limit int) ([]*OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, attempts int) error
}
