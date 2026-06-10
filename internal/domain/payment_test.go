package domain_test

import (
	"testing"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPayment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payment domain.Payment
		wantErr string
	}{
		{
			name: "valid payment",
			payment: domain.Payment{
				ID:         uuid.New(),
				CustomerID: uuid.New(),
				Amount:     100.0,
				Currency:   "USD",
			},
			wantErr: "",
		},
		{
			name: "invalid amount zero",
			payment: domain.Payment{
				ID:         uuid.New(),
				CustomerID: uuid.New(),
				Amount:     0.0,
				Currency:   "USD",
			},
			wantErr: domain.ErrInvalidAmount.Error(),
		},
		{
			name: "invalid amount negative",
			payment: domain.Payment{
				ID:         uuid.New(),
				CustomerID: uuid.New(),
				Amount:     -10.0,
				Currency:   "USD",
			},
			wantErr: domain.ErrInvalidAmount.Error(),
		},
		{
			name: "unsupported currency",
			payment: domain.Payment{
				ID:         uuid.New(),
				CustomerID: uuid.New(),
				Amount:     100.0,
				Currency:   "GBP",
			},
			wantErr: domain.ErrInvalidCurrency.Error(),
		},
		{
			name: "missing customer_id",
			payment: domain.Payment{
				ID:         uuid.New(),
				CustomerID: uuid.Nil,
				Amount:     100.0,
				Currency:   "USD",
			},
			wantErr: "customer_id is required",
		},
		{
			name: "supported currency BRL",
			payment: domain.Payment{
				ID:         uuid.New(),
				CustomerID: uuid.New(),
				Amount:     50.0,
				Currency:   "BRL",
			},
			wantErr: "",
		},
		{
			name: "supported currency EUR",
			payment: domain.Payment{
				ID:         uuid.New(),
				CustomerID: uuid.New(),
				Amount:     50.0,
				Currency:   "EUR",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payment.Validate()
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPayment_StateTransitions(t *testing.T) {
	t.Run("valid transitions from pending", func(t *testing.T) {
		p := &domain.Payment{Status: domain.StatusPending}

		// Pending -> Processing
		assert.True(t, p.CanTransitionTo(domain.StatusProcessing))
		err := p.TransitionTo(domain.StatusProcessing)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusProcessing, p.Status)

		// Pending -> Failed
		p = &domain.Payment{Status: domain.StatusPending}
		assert.True(t, p.CanTransitionTo(domain.StatusFailed))
		err = p.TransitionTo(domain.StatusFailed)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusFailed, p.Status)
	})

	t.Run("valid transitions from processing", func(t *testing.T) {
		p := &domain.Payment{Status: domain.StatusProcessing}
		assert.True(t, p.CanTransitionTo(domain.StatusApproved))
		assert.True(t, p.CanTransitionTo(domain.StatusRejected))
		assert.True(t, p.CanTransitionTo(domain.StatusFailed))

		err := p.TransitionTo(domain.StatusApproved)
		assert.NoError(t, err)
		assert.Equal(t, domain.StatusApproved, p.Status)
	})

	t.Run("invalid transitions from pending", func(t *testing.T) {
		p := &domain.Payment{Status: domain.StatusPending}
		assert.False(t, p.CanTransitionTo(domain.StatusApproved))
		assert.False(t, p.CanTransitionTo(domain.StatusRejected))

		err := p.TransitionTo(domain.StatusApproved)
		assert.ErrorIs(t, err, domain.ErrInvalidStatus)
	})

	t.Run("terminal states are locked", func(t *testing.T) {
		terminalStates := []string{domain.StatusApproved, domain.StatusRejected, domain.StatusFailed}
		for _, state := range terminalStates {
			p := &domain.Payment{Status: state}
			assert.False(t, p.CanTransitionTo(domain.StatusPending))
			assert.False(t, p.CanTransitionTo(domain.StatusProcessing))
			assert.False(t, p.CanTransitionTo(domain.StatusApproved))

			err := p.TransitionTo(domain.StatusProcessing)
			assert.ErrorIs(t, err, domain.ErrInvalidStatus)
		}
	})

	t.Run("invalid state default check", func(t *testing.T) {
		p := &domain.Payment{Status: "UNKNOWN"}
		assert.False(t, p.CanTransitionTo(domain.StatusApproved))
	})
}
