package domain

import "context"

// EventPublisher defines the contract for publishing events asynchronously to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
	Close() error
}
