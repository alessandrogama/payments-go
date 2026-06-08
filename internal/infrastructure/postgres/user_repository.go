package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new instance of UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user record. If email violates unique constraint, returns ErrUserAlreadyExists.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	exec := GetExecutor(ctx, r.db)

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := exec.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user in db: %w", err)
	}

	return nil
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	exec := GetExecutor(ctx, r.db)

	row := exec.QueryRowContext(ctx, query, email)

	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to scan user by email from db: %w", err)
	}

	return &u, nil
}

// GetByID retrieves a user by their unique ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	exec := GetExecutor(ctx, r.db)

	row := exec.QueryRowContext(ctx, query, id)

	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to scan user by id from db: %w", err)
	}

	return &u, nil
}
