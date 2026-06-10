package application

import (
	"context"
	"errors"

	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/internal/domain"
	"github.com/aless/gopay-processing-engine/pkg/security"
	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type authService struct {
	userRepo domain.UserRepository
	cfg      *config.Config
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(userRepo domain.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *authService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashedPassword,
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", errors.New("email and password are required")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", errors.New("invalid email or password")
		}
		return "", err
	}

	if !security.CheckPasswordHash(password, user.PasswordHash) {
		return "", errors.New("invalid email or password")
	}

	token, err := security.GenerateToken(user.ID.String(), user.Email, s.cfg.JWTSecret, s.cfg.JWTExpirationHours)
	if err != nil {
		return "", err
	}

	return token, nil
}
