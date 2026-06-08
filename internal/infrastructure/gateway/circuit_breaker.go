package gateway

import (
	"context"
	"errors"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"
)

type CircuitBreakerGateway struct {
	next    domain.PaymentGateway
	breaker *gobreaker.CircuitBreaker[*domain.GatewayResponse]
}

// NewCircuitBreakerGateway wraps an existing PaymentGateway with a Circuit Breaker.
func NewCircuitBreakerGateway(next domain.PaymentGateway) *CircuitBreakerGateway {
	settings := gobreaker.Settings{
		Name:        "payment-gateway",
		MaxRequests: 3,                 // Maximum requests allowed in half-open state
		Interval:    5 * time.Second,  // Cyclic period to clear failure counts
		Timeout:     10 * time.Second, // Duration of open state before half-opening
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip the breaker if we have 3 consecutive failures
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("Circuit Breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}

	cb := gobreaker.NewCircuitBreaker[*domain.GatewayResponse](settings)

	return &CircuitBreakerGateway{
		next:    next,
		breaker: cb,
	}
}

// Process forwards the payment processing request to the decorated gateway, protected by the circuit breaker.
func (c *CircuitBreakerGateway) Process(ctx context.Context, payment *domain.Payment) (*domain.GatewayResponse, error) {
	resp, err := c.breaker.Execute(func() (*domain.GatewayResponse, error) {
		return c.next.Process(ctx, payment)
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			logger.Error("Circuit Breaker is OPEN, blocking gateway request", zap.String("payment_id", payment.ID.String()))
			return nil, domain.ErrGatewayUnavailable
		}
		return nil, err
	}

	return resp, nil
}
