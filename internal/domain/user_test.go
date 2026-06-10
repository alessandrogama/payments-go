package domain_test

import (
	"testing"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUser_Creation(t *testing.T) {
	id := uuid.New()
	email := "admin@gopay.com"
	hash := "somehashedpassword"
	now := time.Now()

	user := domain.User{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, id, user.ID)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, hash, user.PasswordHash)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
}
