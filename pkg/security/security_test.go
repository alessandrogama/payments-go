package security_test

import (
	"testing"
	"time"

	"github.com/aless/gopay-processing-engine/pkg/security"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestHashAndComparePassword(t *testing.T) {
	password := "supersecure123"

	// 1. Test Hashing
	hash, err := security.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// 2. Test Correct Password Matching
	match := security.CheckPasswordHash(password, hash)
	assert.True(t, match)

	// 3. Test Incorrect Password Matching
	match = security.CheckPasswordHash("wrongpassword", hash)
	assert.False(t, match)
}

func TestJWTTokenGenerationAndValidation(t *testing.T) {
	secret := "test-secret-key-12345"
	userID := "user-123-id"
	email := "user@gopay.com"
	expirationHours := 2

	// 1. Generate Token
	tokenStr, err := security.GenerateToken(userID, email, secret, expirationHours)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	// 2. Validate Valid Token
	claims, err := security.ValidateToken(tokenStr, secret)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, "gopay-processing-engine", claims.Issuer)

	// 3. Validate Token with Wrong Secret
	claimsWrongSecret, err := security.ValidateToken(tokenStr, "wrong-secret-key")
	assert.Error(t, err)
	assert.Nil(t, claimsWrongSecret)
	assert.ErrorIs(t, err, security.ErrInvalidToken)

	// 4. Validate Malformed Token
	claimsMalformed, err := security.ValidateToken("invalid.token.string", secret)
	assert.Error(t, err)
	assert.Nil(t, claimsMalformed)
	assert.ErrorIs(t, err, security.ErrInvalidToken)
}

func TestJWTExpiredToken(t *testing.T) {
	secret := "test-secret-key-12345"
	userID := "user-123-id"
	email := "user@gopay.com"

	// Generate a token that expired 1 hour ago
	claims := security.TokenClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "gopay-processing-engine",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	// Validate expired token
	claimsResult, err := security.ValidateToken(tokenStr, secret)
	assert.Error(t, err)
	assert.Nil(t, claimsResult)
	assert.ErrorIs(t, err, security.ErrExpiredToken)
}
