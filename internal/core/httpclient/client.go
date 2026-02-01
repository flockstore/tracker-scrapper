package httpclient

import (
	"net/http"
	"time"

	"tracker-scrapper/internal/core/logger"

	"go.uber.org/zap"
)

// LoggingRoundTripper captures request details for debugging.
type LoggingRoundTripper struct {
	// Proxied is the underlying RoundTripper to execute the request.
	Proxied http.RoundTripper
}

// RoundTrip executes the request and logs details.
func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	logger.Get().Debug("HTTP Request Started",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
	)

	resp, err := lrt.Proxied.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		logger.Get().Error("HTTP Request Failed",
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Get().Debug("HTTP Request Completed",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration),
	)

	return resp, nil
}

// NewClient returns an http.Client with logging middleware.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &LoggingRoundTripper{
			Proxied: http.DefaultTransport,
		},
		Timeout: timeout,
	}
}
