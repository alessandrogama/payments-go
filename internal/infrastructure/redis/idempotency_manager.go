package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type IdempotencyManager struct {
	client *redis.Client
}

// NewIdempotencyManager creates a new instance of IdempotencyManager.
func NewIdempotencyManager(client *redis.Client) *IdempotencyManager {
	return &IdempotencyManager{client: client}
}

func (m *IdempotencyManager) getCacheKey(key string) string {
	return fmt.Sprintf("idempotency:%s", key)
}

// TryAcquire atomically sets the idempotency key if it does not exist.
// Returns the associated payment ID, true if it already existed, and error if any.
func (m *IdempotencyManager) TryAcquire(ctx context.Context, key string, paymentID uuid.UUID, ttl time.Duration) (uuid.UUID, bool, error) {
	cacheKey := m.getCacheKey(key)

	// SetNX (Set if Not Exists) sets the key only if it does not already exist, atomically.
	success, err := m.client.SetNX(ctx, cacheKey, paymentID.String(), ttl).Result()
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to execute SetNX for idempotency check: %w", err)
	}

	if success {
		// Key did not exist and was successfully stored. This is a new request.
		return paymentID, false, nil
	}

	// Key already existed. Retrieve the associated payment ID.
	val, err := m.client.Get(ctx, cacheKey).Result()
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to fetch existing payment ID for idempotency key: %w", err)
	}

	existingID, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to parse existing UUID from redis: %w", err)
	}

	return existingID, true, nil
}
