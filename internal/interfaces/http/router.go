package http

import (
	_ "github.com/aless/gopay-processing-engine/docs" // Swagger docs package
	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/internal/middleware"
	"github.com/aless/gopay-processing-engine/pkg/telemetry"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter initializes routes and hooks middleware.
func SetupRouter(cfg *config.Config, authHandler *AuthHandler, paymentHandler *PaymentHandler) *gin.Engine {
	r := gin.Default()
	r.Use(telemetry.MetricsMiddleware())

	// Swagger documentation route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
