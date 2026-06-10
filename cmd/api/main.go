package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aless/gopay-processing-engine/internal/application"
	"github.com/aless/gopay-processing-engine/internal/config"
	infraKafka "github.com/aless/gopay-processing-engine/internal/infrastructure/kafka"
	"github.com/aless/gopay-processing-engine/internal/infrastructure/postgres"
	"github.com/aless/gopay-processing-engine/internal/infrastructure/redis"
	httpInterfaces "github.com/aless/gopay-processing-engine/internal/interfaces/http"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/aless/gopay-processing-engine/pkg/telemetry"
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

	logger.Info("GoPay Processing Engine API initializing",
		zap.String("env", cfg.AppEnv),
		zap.String("port", cfg.HTTPServerPort),
	)

	// 2.5. Initialize OTel Tracer
	otelCtx, otelCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer otelCancel()
	shutdownTracer, err := telemetry.InitTracer(otelCtx, cfg.OTELServiceName, cfg.OTELJaegerEndpoint)
	if err != nil {
		logger.Warn("Failed to initialize OTel Tracer", zap.Error(err))
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer shutdownCancel()
			shutdownTracer(shutdownCtx)
		}()
	}

	// 2.6. Start Prometheus scraping server
	metricsSrv := telemetry.StartMetricsServer(cfg.PrometheusPort)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = metricsSrv.Shutdown(shutdownCtx)
	}()

	// 3. Connect to Database
	db, err := postgres.NewConnection(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// 4. Run database migrations
	err = postgres.RunMigrations(db, "migrations")
	if err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}

	// 5. Connect to Redis
	rdb, err := redis.NewClient(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer rdb.Close()

	// 6. Connect to Kafka for Outbox Publishing
	publisher := infraKafka.NewProducer(cfg.KafkaBrokers)
	defer func() {
		if err := publisher.Close(); err != nil {
			logger.Error("Failed to close Kafka producer", zap.Error(err))
		}
	}()

	// 7. Initialize Repositories
	userRepo := postgres.NewUserRepository(db)
	paymentRepo := postgres.NewPaymentRepository(db)
	outboxRepo := postgres.NewOutboxRepository(db)

	// 8. Initialize Cache adapters
	paymentCache := redis.NewPaymentCache(rdb)
	idempotencyManager := redis.NewIdempotencyManager(rdb)

	// 9. Start transactional Outbox processor in background
	outboxProcessor := application.NewOutboxProcessor(outboxRepo, publisher, 1*time.Second, 10)
	outboxCtx, outboxCancel := context.WithCancel(context.Background())
	defer outboxCancel()

	go outboxProcessor.Start(outboxCtx)

	// 10. Initialize Services
	authService := application.NewAuthService(userRepo, cfg)
	paymentService := application.NewPaymentService(paymentRepo, paymentCache, idempotencyManager, outboxRepo, db, cfg.RedisTTL)

	// 11. Initialize HTTP Handlers
	authHandler := httpInterfaces.NewAuthHandler(authService)
	paymentHandler := httpInterfaces.NewPaymentHandler(paymentService)

	// 12. Setup Router
	router := httpInterfaces.SetupRouter(cfg, authHandler, paymentHandler)

	// 13. Setup HTTP Server with graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPServerPort),
		Handler: router,
	}

	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// 14. Catch OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("Shutdown signal received, initiating graceful shutdown", zap.String("signal", sig.String()))

	// Stop outbox background processor
	outboxCancel()

	// Shutdown HTTP Server with 10s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP Server shutdown failed", zap.Error(err))
	} else {
		logger.Info("HTTP Server gracefully stopped")
	}

	logger.Info("GoPay API stopped.")
}
