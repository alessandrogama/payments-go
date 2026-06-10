package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
)

var (
	// HttpRequestsTotal counts HTTP requests processed, labeled by status, path, and method
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"path", "method", "status"},
	)

	// HttpRequestDuration measures HTTP latency per route and method
	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	// PaymentProcessingDuration measures the time to process authorization
	PaymentProcessingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "payment_processing_duration_seconds",
			Help:    "Payment processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// KafkaEventsProcessedTotal counts consumed events by topic and final commit status
	KafkaEventsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_events_processed_total",
			Help: "Total number of Kafka events processed",
		},
		[]string{"topic", "status"},
	)
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(PaymentProcessingDuration)
	prometheus.MustRegister(KafkaEventsProcessedTotal)
}

// InitTracer configures the global OpenTelemetry tracer with OTLP/HTTP exporter.
func InitTracer(ctx context.Context, serviceName, collectorEndpoint string) (func(context.Context), error) {
	// Clean endpoint prefix/suffix (strip http:// and paths)
	endpoint := collectorEndpoint
	if len(endpoint) > 7 && endpoint[:7] == "http://" {
		endpoint = endpoint[7:]
	} else if len(endpoint) > 8 && endpoint[:8] == "https://" {
		endpoint = endpoint[8:]
	}
	for i := 0; i < len(endpoint); i++ {
		if endpoint[i] == '/' {
			endpoint = endpoint[:i]
			break
		}
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	shutdown := func(shutdownCtx context.Context) {
		if err := tp.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown Tracer Provider", zap.Error(err))
		}
	}

	logger.Info("Tracer initialized successfully", zap.String("service_name", serviceName), zap.String("endpoint", endpoint))
	return shutdown, nil
}

// StartMetricsServer opens a background HTTP listener to serve Prometheus metrics.
func StartMetricsServer(port string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	go func() {
		logger.Info("Starting Prometheus metrics server", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Prometheus metrics server failed", zap.Error(err))
		}
	}()

	return srv
}
