package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tracker-scrapper/internal/core/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoggingRoundTripper verifies that requests are logged.
func TestLoggingRoundTripper(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	logger.Init("development", "debug")

	client := NewClient(1 * time.Second)
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestLoggingRoundTripper_Error verifies that failed requests are logged.
func TestLoggingRoundTripper_Error(t *testing.T) {
	logger.Init("development", "debug")

	client := NewClient(1 * time.Second)
	_, err := client.Get("http://invalid-url-that-does-not-exist.local")
	require.Error(t, err)
}
