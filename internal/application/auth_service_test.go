package application_test

import (
	"context"
	"testing"

	"github.com/aless/gopay-processing-engine/internal/application"
	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/internal/domain/mocks"
	"github.com/aless/gopay-processing-engine/pkg/security"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpirationHours: 2}

	t.Run("successful registration", func(t *testing.T) {
		mockRepo := new(mocks.MockUserRepository)
		service := application.NewAuthService(mockRepo, cfg)

		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

		user, err := service.Register(ctx, "admin@gopay.com", "password123")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "admin@gopay.com", user.Email)
		assert.NotEmpty(t, user.PasswordHash)

		mockRepo.AssertExpectations(t)
	})

	t.Run("missing email or password", func(t *testing.T) {
		mockRepo := new(mocks.MockUserRepository)
		service := application.NewAuthService(mockRepo, cfg)

		user, err := service.Register(ctx, "", "password123")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "email and password are required")

		user, err = service.Register(ctx, "admin@gopay.com", "")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "email and password are required")
	})

	t.Run("repository error (e.g. duplicate email)", func(t *testing.T) {
		mockRepo := new(mocks.MockUserRepository)
		service := application.NewAuthService(mockRepo, cfg)

		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(domain.ErrUserAlreadyExists)

		user, err := service.Register(ctx, "duplicate@gopay.com", "password123")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)

		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{JWTSecret: "test-secret-key-for-auth-testing", JWTExpirationHours: 2}

	t.Run("successful login", func(t *testing.T) {
		mockRepo := new(mocks.MockUserRepository)
		service := application.NewAuthService(mockRepo, cfg)

		pwd := "mypassword"
		hashedPwd, err := security.HashPassword(pwd)
		assert.NoError(t, err)
		hashedUser := &domain.User{
			ID:           uuid.New(),
			Email:        "user@gopay.com",
			PasswordHash: hashedPwd,
		}

		mockRepo.On("GetByEmail", ctx, "user@gopay.com").Return(hashedUser, nil)

		token, err := service.Login(ctx, "user@gopay.com", pwd)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := new(mocks.MockUserRepository)
		service := application.NewAuthService(mockRepo, cfg)

		mockRepo.On("GetByEmail", ctx, "notfound@gopay.com").Return((*domain.User)(nil), domain.ErrUserNotFound)

		token, err := service.Login(ctx, "notfound@gopay.com", "anypassword")
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "invalid email or password")

		mockRepo.AssertExpectations(t)
	})

	t.Run("incorrect password", func(t *testing.T) {
		mockRepo := new(mocks.MockUserRepository)
		service := application.NewAuthService(mockRepo, cfg)

		pwd := "correctpassword"
		hashedPwd, err := security.HashPassword(pwd)
		assert.NoError(t, err)
		hashedUser := &domain.User{
			ID:           uuid.New(),
			Email:        "user@gopay.com",
			PasswordHash: hashedPwd,
		}

		mockRepo.On("GetByEmail", ctx, "user@gopay.com").Return(hashedUser, nil)

		token, err := service.Login(ctx, "user@gopay.com", "wrongpassword")
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "invalid email or password")

		mockRepo.AssertExpectations(t)
	})

	t.Run("missing input credentials", func(t *testing.T) {
		mockRepo := new(mocks.MockUserRepository)
		service := application.NewAuthService(mockRepo, cfg)

		token, err := service.Login(ctx, "", "password")
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "email and password are required")
	})
}
