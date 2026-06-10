package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aless/gopay-processing-engine/internal/domain"
	internalHttp "github.com/aless/gopay-processing-engine/internal/interfaces/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService mocks the application.AuthService interface
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful registration", func(t *testing.T) {
		mockService := new(MockAuthService)
		handler := internalHttp.NewAuthHandler(mockService)

		r := gin.Default()
		r.POST("/auth/register", handler.Register)

		user := &domain.User{
			ID:    uuid.New(),
			Email: "test@gopay.com",
		}
		mockService.On("Register", mock.Anything, "test@gopay.com", "password123").Return(user, nil)

		body := map[string]string{
			"email":    "test@gopay.com",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), user.ID.String())
		mockService.AssertExpectations(t)
	})

	t.Run("validation error invalid email", func(t *testing.T) {
		mockService := new(MockAuthService)
		handler := internalHttp.NewAuthHandler(mockService)

		r := gin.Default()
		r.POST("/auth/register", handler.Register)

		body := map[string]string{
			"email":    "not-an-email",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "Register")
	})

	t.Run("conflict already exists", func(t *testing.T) {
		mockService := new(MockAuthService)
		handler := internalHttp.NewAuthHandler(mockService)

		r := gin.Default()
		r.POST("/auth/register", handler.Register)

		mockService.On("Register", mock.Anything, "exists@gopay.com", "password123").
			Return((*domain.User)(nil), domain.ErrUserAlreadyExists)

		body := map[string]string{
			"email":    "exists@gopay.com",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful login", func(t *testing.T) {
		mockService := new(MockAuthService)
		handler := internalHttp.NewAuthHandler(mockService)

		r := gin.Default()
		r.POST("/auth/login", handler.Login)

		token := "my-generated-jwt-token"
		mockService.On("Login", mock.Anything, "test@gopay.com", "password123").Return(token, nil)

		body := map[string]string{
			"email":    "test@gopay.com",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), token)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized invalid credentials", func(t *testing.T) {
		mockService := new(MockAuthService)
		handler := internalHttp.NewAuthHandler(mockService)

		r := gin.Default()
		r.POST("/auth/login", handler.Login)

		mockService.On("Login", mock.Anything, "test@gopay.com", "wrongpwd").
			Return("", errors.New("invalid email or password"))

		body := map[string]string{
			"email":    "test@gopay.com",
			"password": "wrongpwd",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockService.AssertExpectations(t)
	})
}
