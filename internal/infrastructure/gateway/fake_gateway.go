package gateway

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FakeGateway struct {
	successRate int           // e.g. 80 for 80%
	latency     time.Duration // e.g. 150ms
}

// NewFakeGateway creates a new instance of FakeGateway.
func NewFakeGateway(successRate int, latency time.Duration) *FakeGateway {
	return &FakeGateway{
		successRate: successRate,
		latency:     latency,
	}
}

// Process simulates processing a payment via an external gateway.
func (fg *FakeGateway) Process(ctx context.Context, payment *domain.Payment) (*domain.GatewayResponse, error) {
	logger.Info("Starting fake payment gateway processing",
		zap.String("payment_id", payment.ID.String()),
		zap.Float64("amount", payment.Amount),
		zap.String("currency", payment.Currency),
	)

	// Simulate network latency
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(fg.latency):
	}

	// Simulate success/failure using random generation
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	roll := r.Intn(100) + 1 // 1 to 100

	if roll <= fg.successRate {
		logger.Info("Fake payment gateway APPROVED the transaction", zap.String("payment_id", payment.ID.String()))
		return &domain.GatewayResponse{
			Status:        domain.StatusApproved,
			TransactionID: uuid.New().String(),
		}, nil
	}

	// 50% chance of REJECTED vs FAILED for simulated errors
	if r.Intn(2) == 0 {
		logger.Info("Fake payment gateway REJECTED the transaction", zap.String("payment_id", payment.ID.String()))
		return &domain.GatewayResponse{
			Status:       domain.StatusRejected,
			ErrorMessage: "insufficient funds or card decline",
		}, nil
	}

	logger.Warn("Fake payment gateway FAILED with unexpected error", zap.String("payment_id", payment.ID.String()))
	return nil, errors.New("gateway internal connection timeout")
}
