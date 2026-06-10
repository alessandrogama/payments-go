package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type MessageHandler = func(ctx context.Context, key string, value []byte) error

type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer creates a new instance of Kafka Consumer.
func NewConsumer(brokers []string, groupID string, topic string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          brokers,
		GroupID:          groupID,
		Topic:            topic,
		MinBytes:         10e3, // 10KB
		MaxBytes:         10e6, // 10MB
		MaxWait:          1 * time.Second,
		CommitInterval:   0, // manual commit
		StartOffset:      kafka.FirstOffset,
		ReadBackoffMin:   100 * time.Millisecond,
		ReadBackoffMax:   5 * time.Second,
	})
	return &Consumer{reader: reader}
}

// Start runs the message processing loop, executing handler on each message.
func (c *Consumer) Start(ctx context.Context, handler MessageHandler) error {
	logger.Info("Starting Kafka consumer",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Kafka consumer context cancelled, stopping reader")
			return ctx.Err()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				logger.Error("Failed to fetch message from Kafka", zap.Error(err))
				time.Sleep(1 * time.Second) // backoff
				continue
			}

			logger.Debug("Received message from Kafka",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.String("key", string(msg.Key)),
			)

			err = handler(ctx, string(msg.Key), msg.Value)
			if err != nil {
				logger.Error("Failed to process message, not committing offset",
					zap.String("topic", msg.Topic),
					zap.String("key", string(msg.Key)),
					zap.Error(err),
				)
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				logger.Error("Failed to commit message offset to Kafka",
					zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset),
					zap.Error(err),
				)
			}
		}
	}
}

// Close gracefully closes the consumer connection.
func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("failed to close kafka reader: %w", err)
	}
	return nil
}
