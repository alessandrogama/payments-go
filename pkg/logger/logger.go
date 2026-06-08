package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance.
var Log *zap.Logger

// Initialize initializes the global logger instance.
func Initialize(env string) {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	Log, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		// Fallback to default standard logger configuration if Zap fails
		panic("failed to initialize logger: " + err.Error())
	}

	zap.ReplaceGlobals(Log)
}

// Info logs a message at InfoLevel.
func Info(message string, fields ...zap.Field) {
	if Log != nil {
		Log.Info(message, fields...)
	}
}

// Error logs a message at ErrorLevel.
func Error(message string, fields ...zap.Field) {
	if Log != nil {
		Log.Error(message, fields...)
	}
}

// Fatal logs a message at FatalLevel and calls os.Exit(1).
func Fatal(message string, fields ...zap.Field) {
	if Log != nil {
		Log.Fatal(message, fields...)
	} else {
		os.Exit(1)
	}
}

// Debug logs a message at DebugLevel.
func Debug(message string, fields ...zap.Field) {
	if Log != nil {
		Log.Debug(message, fields...)
	}
}

// Warn logs a message at WarnLevel.
func Warn(message string, fields ...zap.Field) {
	if Log != nil {
		Log.Warn(message, fields...)
	}
}

// With creates a child logger with the given fields.
func With(fields ...zap.Field) *zap.Logger {
	if Log != nil {
		return Log.With(fields...)
	}
	return nil
}
