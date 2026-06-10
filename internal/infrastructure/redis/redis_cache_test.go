package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/aless/gopay-processing-engine/internal/domain"
	infraRedis "github.com/aless/gopay-processing-engine/internal/infrastructure/redis"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestPaymentCache_SetAndGet(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	cache := infraRedis.NewPaymentCache(client)
	ctx := context.Background()
	paymentID := uuid.New()

	payment := &domain.Payment{
		ID:         paymentID,
		CustomerID: uuid.New(),
		Amount:     350.75,
		Currency:   "BRL",
		Status:     domain.StatusPending,
	}

	t.Run("cache miss initially", func(t *testing.T) {
		got, err := cache.Get(ctx, paymentID)
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("cache set and hit", func(t *testing.T) {
		err = cache.Set(ctx, payment, 1*time.Hour)
		assert.NoError(t, err)

		got, err := cache.Get(ctx, paymentID)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, payment.Amount, got.Amount)
		assert.Equal(t, payment.Currency, got.Currency)
	})

	t.Run("cache delete", func(t *testing.T) {
		err = cache.Delete(ctx, paymentID)
		assert.NoError(t, err)

		got, err := cache.Get(ctx, paymentID)
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestIdempotencyManager_TryAcquire(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	manager := infraRedis.NewIdempotencyManager(client)
	ctx := context.Background()
	paymentID := uuid.New()
	key := "idemp-key-999"

	t.Run("first acquire - success", func(t *testing.T) {
		gotID, isDup, err := manager.TryAcquire(ctx, key, paymentID, 10*time.Minute)
		assert.NoError(t, err)
		assert.False(t, isDup)
		assert.Equal(t, paymentID, gotID)
	})

	t.Run("second acquire - duplicate", func(t *testing.T) {
		newPaymentID := uuid.New()
		gotID, isDup, err := manager.TryAcquire(ctx, key, newPaymentID, 10*time.Minute)
		assert.NoError(t, err)
		assert.True(t, isDup)
		assert.Equal(t, paymentID, gotID) // Should return the FIRST registered payment ID
	})
}
