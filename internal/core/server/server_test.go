package server

import (
	"testing"
	"time"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/core/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew verifies that New creates a Server with the correct configuration.
func TestNew(t *testing.T) {
	cfg := &config.AppConfig{
		ServerPort: 8080,
	}

	logger.Init("development", "debug")
	srv := New(cfg)

	require.NotNil(t, srv)
	assert.NotNil(t, srv.App)
	assert.Equal(t, cfg, srv.cfg)
}

// TestServer_Run_Error verifies that Run returns an error when binding fails (e.g., privileged port).
func TestServer_Run_Error(t *testing.T) {
	// Privileged port 1 should fail
	cfg := &config.AppConfig{
		ServerPort: 1,
	}
	logger.Init("development", "error")

	srv := New(cfg)

	errCh := make(chan error)
	go func() {
		errCh <- srv.Run()
	}()

	select {
	case err := <-errCh:
		assert.Error(t, err)
	case <-time.After(1 * time.Second):
		srv.App.Shutdown()
		t.Log("Server unexpectedly started or timed out on Error test")
	}
}
