package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aless/gopay-processing-engine/internal/application"
	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/internal/infrastructure/gateway"
	infraKafka "github.com/aless/gopay-processing-engine/internal/infrastructure/kafka"
	"github.com/aless/gopay-processing-engine/internal/infrastructure/postgres"
	"github.com/aless/gopay-processing-engine/internal/infrastructure/redis"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	// 2. Initialize logger
	logger.Initialize(cfg.AppEnv)
	defer func() {
		if logger.Log != nil {
			_ = logger.Log.Sync()
		}
	}()

	logger.Info("GoPay Processing Engine Worker initializing",
		zap.String("env", cfg.AppEnv),
		zap.Strings("kafka_brokers", cfg.KafkaBrokers),
		zap.String("kafka_group", cfg.KafkaGroupID),
	)

	// 3. Connect to PostgreSQL
	db, err := postgres.NewConnection(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer db.Close()

	// 4. Connect to Redis
	rdb, err := redis.NewClient(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer rdb.Close()

	// 5. Connect to Kafka
	publisher := infraKafka.NewProducer(cfg.KafkaBrokers)
	defer func() {
		if err := publisher.Close(); err != nil {
			logger.Error("Failed to close Kafka producer", zap.Error(err))
		}
	}()

	consumer := infraKafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopicCreated)
	defer func() {
		if err := consumer.Close(); err != nil {
			logger.Error("Failed to close Kafka consumer", zap.Error(err))
		}
	}()

	// 6. Instantiate repositories & cache adapters
	paymentRepo := postgres.NewPaymentRepository(db)
	paymentCache := redis.NewPaymentCache(rdb)

	// 7. Instantiate Payment Gateway Client wrapped in Circuit Breaker
	fakeGateway := gateway.NewFakeGateway(cfg.FakeGatewaySuccessRate, cfg.FakeGatewayLatency)
	cbGateway := gateway.NewCircuitBreakerGateway(fakeGateway)

	// 8. Instantiate and configure the Worker
	worker := application.NewPaymentWorker(cfg, consumer, publisher, paymentRepo, paymentCache, cbGateway)

	// 9. Context to control worker shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 10. Start the worker in a background goroutine
	errChan := make(chan error, 1)
	go func() {
		logger.Info("Starting Payment Worker listener")
		if err := worker.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// 11. Catch OS shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Shutdown signal received, initiating graceful shutdown", zap.String("signal", sig.String()))
		cancel() // Signal worker contexts to stop processing and close loops
	case err := <-errChan:
		if err != nil {
			logger.Error("Payment Worker error occurred, initiating shutdown", zap.Error(err))
		}
	}

	// 12. Grace period for worker thread exit
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	logger.Info("Cleanup completed. GoPay Worker stopped.")
	_ = shutdownCtx
}
