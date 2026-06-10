package http

import (
	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes routes and hooks middleware.
func SetupRouter(cfg *config.Config, authHandler *AuthHandler, paymentHandler *PaymentHandler) *gin.Engine {
	r := gin.Default()

	// Public routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes
	payments := r.Group("/payments")
	payments.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		payments.POST("", paymentHandler.Create)
		payments.GET("/:id", paymentHandler.GetByID)
		payments.GET("", paymentHandler.List)
	}

	return r
}
