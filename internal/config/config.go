package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration variables for the application.
type Config struct {
	AppEnv         string
	HTTPServerPort string

	// Database configurations
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis configurations
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisTTL      time.Duration

	// Kafka configurations
	KafkaBrokers        []string
	KafkaGroupID        string
	KafkaTopicCreated   string
	KafkaTopicProcessed string
	KafkaTopicFailed    string
	KafkaTopicDLQ       string

	// Security configurations
	JWTSecret          string
	JWTExpirationHours int

	// Telemetry configurations
	OTELServiceName    string
	OTELJaegerEndpoint string
	PrometheusPort     string

	// Gateway configurations
	FakeGatewaySuccessRate int           // e.g. 80 for 80%
	FakeGatewayLatency     time.Duration // e.g. 100ms
}

// LoadConfig loads configuration from environment variables and .env file.
func LoadConfig() (*Config, error) {
	// Attempt to load .env file if it exists, ignore error if it doesn't
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		HTTPServerPort:         getEnv("HTTP_SERVER_PORT", "8080"),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBUser:                 getEnv("DB_USER", "postgres"),
		DBPassword:             getEnv("DB_PASSWORD", "postgres"),
		DBName:                 getEnv("DB_NAME", "gopay"),
		DBSSLMode:              getEnv("DB_SSL_MODE", "disable"),
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		RedisDB:                getEnvAsInt("REDIS_DB", 0),
		RedisTTL:               time.Duration(getEnvAsInt("REDIS_TTL_SECONDS", 3600)) * time.Second,
		KafkaBrokers:           []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		KafkaGroupID:           getEnv("KAFKA_GROUP_ID", "gopay-workers"),
		KafkaTopicCreated:      getEnv("KAFKA_TOPIC_CREATED", "payments.created"),
		KafkaTopicProcessed:    getEnv("KAFKA_TOPIC_PROCESSED", "payments.processed"),
		KafkaTopicFailed:       getEnv("KAFKA_TOPIC_FAILED", "payments.failed"),
		KafkaTopicDLQ:          getEnv("KAFKA_TOPIC_DLQ", "payments.dlq"),
		JWTSecret:              getEnv("JWT_SECRET", "super-secret-key-change-it-in-production"),
		JWTExpirationHours:     getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		OTELServiceName:        getEnv("OTEL_SERVICE_NAME", "gopay-processing-engine"),
		OTELJaegerEndpoint:     getEnv("OTEL_JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
		PrometheusPort:         getEnv("PROMETHEUS_PORT", "2112"),
		FakeGatewaySuccessRate: getEnvAsInt("FAKE_GATEWAY_SUCCESS_RATE", 80),
		FakeGatewayLatency:     time.Duration(getEnvAsInt("FAKE_GATEWAY_LATENCY_MS", 150)) * time.Millisecond,
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: env %s=%s is not a valid integer, falling back to default %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
