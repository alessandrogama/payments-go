package main

import (
	"fmt"

	"github.com/aless/gopay-processing-engine/internal/config"
	"github.com/aless/gopay-processing-engine/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	// Initialize logger
	logger.Initialize(cfg.AppEnv)
	defer func() {
		if logger.Log != nil {
			_ = logger.Log.Sync()
		}
	}()

	logger.Info("GoPay Processing Engine initialized successfully",
		zap.String("env", cfg.AppEnv),
		zap.String("port", cfg.HTTPServerPort),
	)
}
