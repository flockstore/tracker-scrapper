package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestInit verifies logger initialization for different environments.
func TestInit(t *testing.T) {
	t.Run("Development", func(t *testing.T) {
		err := Init("development", "debug")
		require.NoError(t, err)
		assert.NotNil(t, globalLogger)
		assert.True(t, globalLogger.Core().Enabled(zap.DebugLevel))
	})

	t.Run("Production", func(t *testing.T) {
		err := Init("production", "info")
		require.NoError(t, err)
		assert.NotNil(t, globalLogger)
		assert.False(t, globalLogger.Core().Enabled(zap.DebugLevel))
		assert.True(t, globalLogger.Core().Enabled(zap.InfoLevel))
	})

	t.Run("InvalidLevel", func(t *testing.T) {
		err := Init("development", "invalid_level")
		require.NoError(t, err)
	})

	t.Run("BuildError", func(t *testing.T) {
	})
}

// TestGet verifies that Get returns the global logger.
func TestGet(t *testing.T) {
	globalLogger = nil
	assert.NotNil(t, Get())

	Init("development", "info")
	assert.NotNil(t, Get())
	assert.NotEqual(t, zap.NewNop(), Get())
}

// TestSync verifies that Sync does not panic even if logger is nil.
func TestSync(t *testing.T) {
	globalLogger = nil
	Sync()

	Init("development", "info")
	Sync()
}
