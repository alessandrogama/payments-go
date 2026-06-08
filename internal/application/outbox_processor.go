package application

import (
	"context"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"go.uber.org/zap"
)

type OutboxProcessor struct {
	outboxRepo domain.OutboxRepository
	publisher  domain.EventPublisher
	interval   time.Duration
	limit      int
}

// NewOutboxProcessor creates a new instance of OutboxProcessor.
func NewOutboxProcessor(
	outboxRepo domain.OutboxRepository,
	publisher domain.EventPublisher,
	interval time.Duration,
	limit int,
) *OutboxProcessor {
	return &OutboxProcessor{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		interval:   interval,
		limit:      limit,
	}
}

// Start initiates the polling loop for pending outbox events.
func (p *OutboxProcessor) Start(ctx context.Context) {
	logger.Info("Starting Outbox Processor loop",
		zap.Duration("interval", p.interval),
		zap.Int("limit", p.limit),
	)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Outbox Processor loop")
			return
		case <-ticker.C:
			p.processEvents(ctx)
		}
	}
}

func (p *OutboxProcessor) processEvents(ctx context.Context) {
	events, err := p.outboxRepo.GetPending(ctx, p.limit)
	if err != nil {
		logger.Error("Outbox Processor: failed to fetch pending events", zap.Error(err))
		return
	}

	if len(events) == 0 {
		return
	}

	logger.Info("Outbox Processor: processing pending events", zap.Int("count", len(events)))

	for _, event := range events {
		// Use EventType as topic, event ID string as key
		err = p.publisher.Publish(ctx, event.EventType, event.ID.String(), event.Payload)
		if err != nil {
			logger.Error("Outbox Processor: failed to publish event, marking attempt",
				zap.String("event_id", event.ID.String()),
				zap.Error(err),
			)

			attempts := event.Attempts + 1
			if err := p.outboxRepo.MarkFailed(ctx, event.ID, attempts); err != nil {
				logger.Error("Outbox Processor: failed to mark event as failed",
					zap.String("event_id", event.ID.String()),
					zap.Error(err),
				)
			}
			continue
		}

		logger.Info("Outbox Processor: successfully published event", zap.String("event_id", event.ID.String()))

		if err := p.outboxRepo.MarkProcessed(ctx, event.ID); err != nil {
			logger.Error("Outbox Processor: failed to mark event as processed",
				zap.String("event_id", event.ID.String()),
				zap.Error(err),
			)
		}
	}
}
