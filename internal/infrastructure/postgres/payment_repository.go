package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/google/uuid"
)

type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository creates a new instance of PaymentRepository.
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts a new payment record into the database.
func (r *PaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, customer_id, amount, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	exec := GetExecutor(ctx, r.db)

	now := time.Now()
	payment.CreatedAt = now
	payment.UpdatedAt = now

	_, err := exec.ExecContext(ctx, query,
		payment.ID,
		payment.CustomerID,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create payment in db: %w", err)
	}

	return nil
}

// GetByID retrieves a payment by its unique ID.
func (r *PaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	query := `
		SELECT id, customer_id, amount, currency, status, created_at, updated_at
		FROM payments
		WHERE id = $1
	`
	exec := GetExecutor(ctx, r.db)

	row := exec.QueryRowContext(ctx, query, id)

	var p domain.Payment
	err := row.Scan(
		&p.ID,
		&p.CustomerID,
		&p.Amount,
		&p.Currency,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to scan payment from db: %w", err)
	}

	return &p, nil
}

// Update updates an existing payment's status and updated_at time.
func (r *PaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	query := `
		UPDATE payments
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	exec := GetExecutor(ctx, r.db)

	payment.UpdatedAt = time.Now()

	result, err := exec.ExecContext(ctx, query,
		payment.Status,
		payment.UpdatedAt,
		payment.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update payment in db: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

// List retrieves all payment records ordered by creation date descending.
func (r *PaymentRepository) List(ctx context.Context) ([]*domain.Payment, error) {
	query := `
		SELECT id, customer_id, amount, currency, status, created_at, updated_at
		FROM payments
		ORDER BY created_at DESC
	`
	exec := GetExecutor(ctx, r.db)

	rows, err := exec.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		err := rows.Scan(
			&p.ID,
			&p.CustomerID,
			&p.Amount,
			&p.Currency,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return payments, nil
}
