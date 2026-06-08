package domain

import (
	"context"
	"errors"
)

var (
	ErrGatewayUnavailable = errors.New("payment gateway is currently unavailable")
)

// GatewayResponse represents the feedback from a third-party payment gateway.
type GatewayResponse struct {
	Status        string // APPROVED, REJECTED, FAILED
	TransactionID string
	ErrorMessage  string
}

// PaymentGateway defines the contract to communicate with external payment providers.
type PaymentGateway interface {
	Process(ctx context.Context, payment *Payment) (*GatewayResponse, error)
}
