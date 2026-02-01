package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.Logger

// Init initializes the global logger.
// For "development" env, it produces pretty console logs.
// For "production" env, it (usually) produces JSON logs.
func Init(environment string, level string) error {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	l, err := zapcore.ParseLevel(level)
	if err == nil {
		config.Level = zap.NewAtomicLevelAt(l)
	}

	logger, err := config.Build()
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// Get returns the global logger instance.
// If not initialized, it returns a no-op logger to prevent panics.
func Get() *zap.Logger {
	if globalLogger == nil {
		return zap.NewNop()
	}
	return globalLogger
}

// Sync flushes any buffered log entries.
func Sync() {
	if globalLogger != nil {
		globalLogger.Sync()
	}
}
