package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

type CreatePaymentInput struct {
	CustomerID     uuid.UUID
	Amount         float64
	Currency       string
	IdempotencyKey string
}

type PaymentService interface {
	CreatePayment(ctx context.Context, input CreatePaymentInput) (*domain.Payment, error)
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	ListPayments(ctx context.Context) ([]*domain.Payment, error)
}

type paymentService struct {
	paymentRepo        domain.PaymentRepository
	paymentCache       domain.PaymentCache
	idempotencyManager domain.IdempotencyManager
	outboxRepo         domain.OutboxRepository
	db                 *sql.DB
	redisTTL           time.Duration
}

// NewPaymentService creates a new instance of PaymentService.
func NewPaymentService(
	paymentRepo domain.PaymentRepository,
	paymentCache domain.PaymentCache,
	idempotencyManager domain.IdempotencyManager,
	outboxRepo domain.OutboxRepository,
	db *sql.DB,
	redisTTL time.Duration,
) PaymentService {
	return &paymentService{
		paymentRepo:        paymentRepo,
		paymentCache:       paymentCache,
		idempotencyManager: idempotencyManager,
		outboxRepo:         outboxRepo,
		db:                 db,
		redisTTL:           redisTTL,
	}
}

func (s *paymentService) CreatePayment(ctx context.Context, input CreatePaymentInput) (*domain.Payment, error) {
	if input.IdempotencyKey == "" {
		return nil, errors.New("idempotency key is required")
	}

	paymentID := uuid.New()

	// 1. Try to acquire the idempotency key in Redis
	actualID, isDuplicate, err := s.idempotencyManager.TryAcquire(ctx, input.IdempotencyKey, paymentID, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	if isDuplicate {
		// Try fetching from cache first
		cachedPayment, err := s.paymentCache.Get(ctx, actualID)
		if err == nil && cachedPayment != nil {
			return cachedPayment, nil
		}

		// Fallback to DB
		dbPayment, err := s.paymentRepo.GetByID(ctx, actualID)
		if err != nil {
			if errors.Is(err, domain.ErrPaymentNotFound) {
				return nil, domain.ErrIdempotencyFailed
			}
			return nil, err
		}

		// Update cache
		_ = s.paymentCache.Set(ctx, dbPayment, s.redisTTL)
		return dbPayment, nil
	}

	// 2. Create and validate domain payment entity
	payment := &domain.Payment{
		ID:         paymentID,
		CustomerID: input.CustomerID,
		Amount:     input.Amount,
		Currency:   input.Currency,
		Status:     domain.StatusPending,
	}

	if err := payment.Validate(); err != nil {
		return nil, err
	}

	// 3. Serialize payment for the transactional outbox event
	payload, err := json.Marshal(payment)
	if err != nil {
		return nil, err
	}

	outboxEvent := &domain.OutboxEvent{
		ID:        uuid.New(),
		EventType: "payments.created",
		Payload:   payload,
	}

	// 4. Save payment and outbox event atomically inside a DB transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txCtx := postgres.WithTx(ctx, tx)

	if err := s.paymentRepo.Create(txCtx, payment); err != nil {
		return nil, err
	}

	if err := s.outboxRepo.Save(txCtx, outboxEvent); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// 5. Populate cache
	_ = s.paymentCache.Set(ctx, payment, s.redisTTL)

	return payment, nil
}

func (s *paymentService) GetPaymentByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	// Try fetching from Redis cache
	cachedPayment, err := s.paymentCache.Get(ctx, id)
	if err == nil && cachedPayment != nil {
		return cachedPayment, nil
	}

	// Fallback to database query
	payment, err := s.paymentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result for subsequent requests
	_ = s.paymentCache.Set(ctx, payment, s.redisTTL)

	return payment, nil
}

func (s *paymentService) ListPayments(ctx context.Context) ([]*domain.Payment, error) {
	return s.paymentRepo.List(ctx)
}
