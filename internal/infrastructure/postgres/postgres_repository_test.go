package postgres_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestPaymentRepository_Create(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPaymentRepository(db)
	ctx := context.Background()

	payment := &domain.Payment{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		Amount:     120.50,
		Currency:   "USD",
		Status:     domain.StatusPending,
	}

	query := `INSERT INTO payments (id, customer_id, amount, currency, status, created_at, updated_at)`
	sqlMock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs(
			payment.ID,
			payment.CustomerID,
			payment.Amount,
			payment.Currency,
			payment.Status,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, payment)
	assert.NoError(t, err)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPaymentRepository_GetByID(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPaymentRepository(db)
	ctx := context.Background()
	paymentID := uuid.New()

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "customer_id", "amount", "currency", "status", "created_at", "updated_at"}).
			AddRow(paymentID, uuid.New(), 100.0, "USD", domain.StatusApproved, time.Now(), time.Now())

		query := `SELECT id, customer_id, amount, currency, status, created_at, updated_at FROM payments WHERE id = $1`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(paymentID).
			WillReturnRows(rows)

		payment, err := repo.GetByID(ctx, paymentID)
		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, paymentID, payment.ID)
		assert.Equal(t, domain.StatusApproved, payment.Status)
	})

	t.Run("not found", func(t *testing.T) {
		query := `SELECT id, customer_id, amount, currency, status, created_at, updated_at FROM payments WHERE id = $1`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(paymentID).
			WillReturnError(sql.ErrNoRows)

		payment, err := repo.GetByID(ctx, paymentID)
		assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
		assert.Nil(t, payment)
	})
}

func TestPaymentRepository_Update(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPaymentRepository(db)
	ctx := context.Background()
	paymentID := uuid.New()

	payment := &domain.Payment{
		ID:     paymentID,
		Status: domain.StatusApproved,
	}

	t.Run("successful update", func(t *testing.T) {
		query := `UPDATE payments SET status = $1, updated_at = $2 WHERE id = $3`
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(payment.Status, sqlmock.AnyArg(), payment.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Update(ctx, payment)
		assert.NoError(t, err)
	})

	t.Run("payment not found to update", func(t *testing.T) {
		query := `UPDATE payments SET status = $1, updated_at = $2 WHERE id = $3`
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(payment.Status, sqlmock.AnyArg(), payment.ID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = repo.Update(ctx, payment)
		assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})
}

func TestPaymentRepository_List(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := postgres.NewPaymentRepository(db)
	ctx := context.Background()

	payment1 := uuid.New()
	payment2 := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "amount", "currency", "status", "created_at", "updated_at"}).
		AddRow(payment1, uuid.New(), 100.0, "USD", domain.StatusApproved, time.Now(), time.Now()).
		AddRow(payment2, uuid.New(), 50.0, "EUR", domain.StatusPending, time.Now(), time.Now())

	query := `SELECT id, customer_id, amount, currency, status, created_at, updated_at FROM payments ORDER BY created_at DESC`
	sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

	payments, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, payments, 2)
	assert.Equal(t, payment1, payments[0].ID)
	assert.Equal(t, payment2, payments[1].ID)
}

func TestUserRepository_Queries(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := postgres.NewUserRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	email := "user@gopay.com"

	user := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: "hashedpwd",
	}

	t.Run("Create success", func(t *testing.T) {
		query := `INSERT INTO users (id, email, password_hash, created_at, updated_at)`
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(user.ID, user.Email, user.PasswordHash, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(ctx, user)
		assert.NoError(t, err)
	})

	t.Run("Create duplicate conflict", func(t *testing.T) {
		query := `INSERT INTO users (id, email, password_hash, created_at, updated_at)`
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(user.ID, user.Email, user.PasswordHash, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err = repo.Create(ctx, user)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})

	t.Run("GetByEmail found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at", "updated_at"}).
			AddRow(userID, email, "hashedpwd", time.Now(), time.Now())

		query := `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(email).WillReturnRows(rows)

		got, err := repo.GetByEmail(ctx, email)
		assert.NoError(t, err)
		assert.Equal(t, userID, got.ID)
		assert.Equal(t, email, got.Email)
	})

	t.Run("GetByEmail not found", func(t *testing.T) {
		query := `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(email).WillReturnError(sql.ErrNoRows)

		got, err := repo.GetByEmail(ctx, email)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, got)
	})

	t.Run("GetByID found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at", "updated_at"}).
			AddRow(userID, email, "hashedpwd", time.Now(), time.Now())

		query := `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(userID).WillReturnRows(rows)

		got, err := repo.GetByID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, userID, got.ID)
	})

	t.Run("GetByID not found", func(t *testing.T) {
		query := `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(userID).WillReturnError(sql.ErrNoRows)

		got, err := repo.GetByID(ctx, userID)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, got)
	})
}

func TestOutboxRepository_Queries(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := postgres.NewOutboxRepository(db)
	ctx := context.Background()
	eventID := uuid.New()

	event := &domain.OutboxEvent{
		ID:        eventID,
		EventType: "payments.created",
		Payload:   []byte(`{"amount":10}`),
	}

	t.Run("Save success", func(t *testing.T) {
		query := `INSERT INTO outbox_events (id, event_type, payload, status, attempts, created_at)`
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(event.ID, event.EventType, event.Payload, "PENDING", 0, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Save(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("GetPending success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "event_type", "payload", "status", "attempts", "created_at", "processed_at"}).
			AddRow(eventID, "payments.created", []byte(`{"amount":10}`), "PENDING", 0, time.Now(), nil)

		query := `SELECT id, event_type, payload, status, attempts, created_at, processed_at FROM outbox_events WHERE status = $1 ORDER BY created_at ASC LIMIT $2`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("PENDING", 10).WillReturnRows(rows)

		events, err := repo.GetPending(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, eventID, events[0].ID)
	})

	t.Run("MarkProcessed success", func(t *testing.T) {
		query := `UPDATE outbox_events SET status = $1, processed_at = $2 WHERE id = $3`
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs("PROCESSED", sqlmock.AnyArg(), eventID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.MarkProcessed(ctx, eventID)
		assert.NoError(t, err)
	})

	t.Run("MarkFailed success", func(t *testing.T) {
		query := `UPDATE outbox_events SET status = $1, attempts = $2 WHERE id = $3`
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs("PENDING", 2, eventID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.MarkFailed(ctx, eventID, 2)
		assert.NoError(t, err)
	})
}
