package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PaymentCache struct {
	client *redis.Client
}

// NewPaymentCache creates a new instance of PaymentCache.
func NewPaymentCache(client *redis.Client) *PaymentCache {
	return &PaymentCache{client: client}
}

func (c *PaymentCache) getCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("payment:%s", id.String())
}

// Get fetches a payment from Redis.
func (c *PaymentCache) Get(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	key := c.getCacheKey(id)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Cache miss, return nil but no error (standard cache behavior)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment from redis cache: %w", err)
	}

	var p domain.Payment
	if err := json.Unmarshal([]byte(val), &p); err != nil {
		return nil, fmt.Errorf("failed to deserialize payment: %w", err)
	}

	return &p, nil
}

// Set stores a payment in Redis with a specified TTL.
func (c *PaymentCache) Set(ctx context.Context, payment *domain.Payment, ttl time.Duration) error {
	key := c.getCacheKey(payment.ID)

	data, err := json.Marshal(payment)
	if err != nil {
		return fmt.Errorf("failed to serialize payment: %w", err)
	}

	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to save payment in redis cache: %w", err)
	}

	return nil
}

// Delete removes a payment from the cache (e.g. after updating it).
func (c *PaymentCache) Delete(ctx context.Context, id uuid.UUID) error {
	key := c.getCacheKey(id)
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete payment from redis cache: %w", err)
	}
	return nil
}
