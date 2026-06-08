package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/google/uuid"
)

type OutboxRepository struct {
	db *sql.DB
}

// NewOutboxRepository creates a new instance of OutboxRepository.
func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// Save persists an event inside the outbox table.
func (r *OutboxRepository) Save(ctx context.Context, event *domain.OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, event_type, payload, status, attempts, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	exec := GetExecutor(ctx, r.db)

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.CreatedAt = time.Now()
	event.Status = domain.OutboxStatusPending
	event.Attempts = 0

	_, err := exec.ExecContext(ctx, query,
		event.ID,
		event.EventType,
		event.Payload,
		event.Status,
		event.Attempts,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}

	return nil
}

// GetPending fetches up to 'limit' pending outbox events.
func (r *OutboxRepository) GetPending(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	query := `
		SELECT id, event_type, payload, status, attempts, created_at, processed_at
		FROM outbox_events
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`
	exec := GetExecutor(ctx, r.db)

	rows, err := exec.QueryContext(ctx, query, domain.OutboxStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending outbox events: %w", err)
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		var event domain.OutboxEvent
		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Payload,
			&event.Status,
			&event.Attempts,
			&event.CreatedAt,
			&event.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return events, nil
}

// MarkProcessed updates the status of an event to PROCESSED and sets processed_at.
func (r *OutboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE outbox_events
		SET status = $1, processed_at = $2
		WHERE id = $3
	`
	exec := GetExecutor(ctx, r.db)

	now := time.Now()
	_, err := exec.ExecContext(ctx, query, domain.OutboxStatusProcessed, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as processed: %w", err)
	}

	return nil
}

// MarkFailed increments the attempts and marks status as FAILED if max retries reached.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, attempts int) error {
	status := domain.OutboxStatusPending
	if attempts >= 3 {
		status = domain.OutboxStatusFailed
	}

	query := `
		UPDATE outbox_events
		SET status = $1, attempts = $2
		WHERE id = $3
	`
	exec := GetExecutor(ctx, r.db)

	_, err := exec.ExecContext(ctx, query, status, attempts, id)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as failed: %w", err)
	}

	return nil
}
