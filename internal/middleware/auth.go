package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/aless/gopay-processing-engine/pkg/logger"
	"github.com/aless/gopay-processing-engine/pkg/security"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware creates a Gin middleware that validates the JWT token in Authorization header.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Request missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Warn("Invalid Authorization format, expected Bearer <token>")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format. Expected 'Bearer <token>'"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := security.ValidateToken(tokenStr, jwtSecret)
		if err != nil {
			logger.Warn("JWT validation failed", zap.Error(err))
			if errors.Is(err, security.ErrExpiredToken) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		// Store user details in the context for handlers
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}
