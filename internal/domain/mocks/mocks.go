package mocks

import (
	"context"
	"time"

	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// MockPaymentRepository is a mock implementation of PaymentRepository
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) List(ctx context.Context) ([]*domain.Payment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Payment), args.Error(1)
}

// MockOutboxRepository is a mock implementation of OutboxRepository
type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) Save(ctx context.Context, event *domain.OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetPending(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.OutboxEvent), args.Error(1)
}

func (m *MockOutboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, attempts int) error {
	args := m.Called(ctx, id, attempts)
	return args.Error(0)
}

// MockPaymentGateway is a mock implementation of PaymentGateway
type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) Process(ctx context.Context, payment *domain.Payment) (*domain.GatewayResponse, error) {
	args := m.Called(ctx, payment)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GatewayResponse), args.Error(1)
}

// MockEventPublisher is a mock implementation of EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	args := m.Called(ctx, topic, key, payload)
	return args.Error(0)
}

func (m *MockEventPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockPaymentCache is a mock implementation of PaymentCache
type MockPaymentCache struct {
	mock.Mock
}

func (m *MockPaymentCache) Get(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *MockPaymentCache) Set(ctx context.Context, payment *domain.Payment, ttl time.Duration) error {
	args := m.Called(ctx, payment, ttl)
	return args.Error(0)
}

func (m *MockPaymentCache) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockIdempotencyManager is a mock implementation of IdempotencyManager
type MockIdempotencyManager struct {
	mock.Mock
}

func (m *MockIdempotencyManager) TryAcquire(ctx context.Context, key string, paymentID uuid.UUID, ttl time.Duration) (uuid.UUID, bool, error) {
	args := m.Called(ctx, key, paymentID, ttl)
	return args.Get(0).(uuid.UUID), args.Bool(1), args.Error(2)
}

// MockKafkaConsumer is a mock implementation of KafkaConsumer
type MockKafkaConsumer struct {
	mock.Mock
}

func (m *MockKafkaConsumer) Start(ctx context.Context, handler func(ctx context.Context, key string, value []byte) error) error {
	args := m.Called(ctx, handler)
	return args.Error(0)
}

func (m *MockKafkaConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}
