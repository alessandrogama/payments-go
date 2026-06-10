package application

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/aless/gopay-processing-engine/pkg/telemetry"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type KafkaConsumer interface {
	Start(ctx context.Context, handler func(ctx context.Context, key string, value []byte) error) error
	Close() error
}

type PaymentWorker struct {
	cfg          *config.Config
	consumer     KafkaConsumer
	publisher    domain.EventPublisher
	paymentRepo  domain.PaymentRepository
	paymentCache domain.PaymentCache
	gateway      domain.PaymentGateway
	redisTTL     time.Duration
}

// NewPaymentWorker creates a new instance of PaymentWorker.
func NewPaymentWorker(
	cfg *config.Config,
	consumer KafkaConsumer,
	publisher domain.EventPublisher,
	paymentRepo domain.PaymentRepository,
	paymentCache domain.PaymentCache,
	gateway domain.PaymentGateway,
) *PaymentWorker {
	return &PaymentWorker{
		cfg:          cfg,
		consumer:     consumer,
		publisher:    publisher,
		paymentRepo:  paymentRepo,
		paymentCache: paymentCache,
		gateway:      gateway,
		redisTTL:     cfg.RedisTTL,
	}
}

// Start launches the consumer loop.
func (w *PaymentWorker) Start(ctx context.Context) error {
	return w.consumer.Start(ctx, w.ProcessEvent)
}

// ProcessEvent parses the message, updates state, and runs payment authorization with retries.
func (w *PaymentWorker) ProcessEvent(ctx context.Context, key string, value []byte) error {
	startTime := time.Now()
	logger.Info("Worker received payment event", zap.String("key", key))

	// 1. Unmarshal the payment payload
	var evtPayment domain.Payment
	if err := json.Unmarshal(value, &evtPayment); err != nil {
		logger.Error("Worker received corrupt payload, routing to DLQ", zap.Error(err))
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "corrupt").Inc()
		w.routeToDLQ(ctx, key, value, "corrupt_payload: "+err.Error())
		return nil // Return nil so offset is committed and queue is not blocked
	}

	// 2. Fetch fresh payment state from Postgres
	payment, err := w.paymentRepo.GetByID(ctx, evtPayment.ID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			logger.Error("Worker: payment not found in database, routing to DLQ", zap.String("payment_id", evtPayment.ID.String()))
			telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "not_found").Inc()
			w.routeToDLQ(ctx, key, value, "payment_not_found")
			return nil
		}
		logger.Error("Worker: database error fetching payment", zap.String("payment_id", evtPayment.ID.String()), zap.Error(err))
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "db_error").Inc()
		return err // Retry reading this event from Kafka
	}

	// 3. Ensure payment is in PENDING state
	if payment.Status != domain.StatusPending {
		logger.Warn("Worker: payment is not pending, skipping processing",
			zap.String("payment_id", payment.ID.String()),
			zap.String("current_status", payment.Status),
		)
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "skipped").Inc()
		return nil
	}

	// 4. Transition status to PROCESSING
	if err := payment.TransitionTo(domain.StatusProcessing); err != nil {
		logger.Error("Worker: failed to transition payment to PROCESSING state", zap.String("payment_id", payment.ID.String()), zap.Error(err))
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "state_error").Inc()
		return nil
	}

	if err := w.updatePaymentState(ctx, payment); err != nil {
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "db_error").Inc()
		return err
	}

	// 5. Call Gateway with retry policy
	var gatewayResp *domain.GatewayResponse
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		gatewayResp, err = w.gateway.Process(ctx, payment)
		if err == nil {
			break
		}

		logger.Warn("Worker: payment gateway attempt failed",
			zap.String("payment_id", payment.ID.String()),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		if attempt < maxRetries {
			// Exponential backoff: 500ms, 1s, 2s...
			backoff := time.Duration(math.Pow(2, float64(attempt-1))*500) * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	// 6. Handle authorization outcome
	if err != nil {
		// All retries failed (transient gateway timeout / system error)
		logger.Error("Worker: payment authorization failed after max retries",
			zap.String("payment_id", payment.ID.String()),
			zap.Error(err),
		)

		_ = payment.TransitionTo(domain.StatusFailed)
		_ = w.updatePaymentState(ctx, payment)
		w.publishOutcome(ctx, w.cfg.KafkaTopicFailed, payment)
		w.routeToDLQ(ctx, key, value, "max_retries_exceeded: "+err.Error())
		
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "failed").Inc()
		telemetry.PaymentProcessingDuration.Observe(time.Since(startTime).Seconds())
		return nil
	}

	// Gateway successfully responded
	if gatewayResp.Status == domain.StatusApproved {
		logger.Info("Worker: payment APPROVED by gateway", zap.String("payment_id", payment.ID.String()))
		_ = payment.TransitionTo(domain.StatusApproved)
		_ = w.updatePaymentState(ctx, payment)
		w.publishOutcome(ctx, w.cfg.KafkaTopicProcessed, payment)
		
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "approved").Inc()
	} else {
		// Status is REJECTED
		logger.Info("Worker: payment REJECTED by gateway", zap.String("payment_id", payment.ID.String()), zap.String("reason", gatewayResp.ErrorMessage))
		_ = payment.TransitionTo(domain.StatusRejected)
		_ = w.updatePaymentState(ctx, payment)
		w.publishOutcome(ctx, w.cfg.KafkaTopicFailed, payment)
		
		telemetry.KafkaEventsProcessedTotal.WithLabelValues(w.cfg.KafkaTopicCreated, "rejected").Inc()
	}

	telemetry.PaymentProcessingDuration.Observe(time.Since(startTime).Seconds())
	return nil
}

func (w *PaymentWorker) updatePaymentState(ctx context.Context, payment *domain.Payment) error {
	if err := w.paymentRepo.Update(ctx, payment); err != nil {
		logger.Error("Worker: failed to update payment in database", zap.String("payment_id", payment.ID.String()), zap.Error(err))
		return err
	}

	// Evict/Update Cache
	if err := w.paymentCache.Set(ctx, payment, w.redisTTL); err != nil {
		logger.Warn("Worker: failed to update payment cache", zap.String("payment_id", payment.ID.String()), zap.Error(err))
	}

	return nil
}

func (w *PaymentWorker) publishOutcome(ctx context.Context, topic string, payment *domain.Payment) {
	payload, err := json.Marshal(payment)
	if err != nil {
		logger.Error("Worker: failed to serialize outcome payment", zap.Error(err))
		return
	}

	err = w.publisher.Publish(ctx, topic, payment.ID.String(), payload)
	if err != nil {
		logger.Error("Worker: failed to publish outcome event", zap.String("topic", topic), zap.Error(err))
	}
}

func (w *PaymentWorker) routeToDLQ(ctx context.Context, key string, value []byte, reason string) {
	dlqPayload := struct {
		Reason    string    `json:"reason"`
		Timestamp time.Time `json:"timestamp"`
		Payload   []byte    `json:"payload"`
	}{
		Reason:    reason,
		Timestamp: time.Now(),
		Payload:   value,
	}

	bytes, err := json.Marshal(dlqPayload)
	if err != nil {
		logger.Error("Worker: failed to serialize DLQ message wrapper", zap.Error(err))
		return
	}

	dlqKey := key
	if dlqKey == "" {
		dlqKey = uuid.New().String()
	}

	err = w.publisher.Publish(ctx, w.cfg.KafkaTopicDLQ, dlqKey, bytes)
	if err != nil {
		logger.Error("Worker: failed to route message to DLQ topic", zap.String("topic", w.cfg.KafkaTopicDLQ), zap.Error(err))
	} else {
		logger.Warn("Worker: routed bad/failed message to DLQ", zap.String("reason", reason))
	}
}
