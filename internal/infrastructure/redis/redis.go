package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewClient initializes a connection to the Redis server.
func NewClient(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Successfully connected to Redis", zap.String("addr", cfg.RedisAddr), zap.Int("db", cfg.RedisDB))
	return rdb, nil
}
