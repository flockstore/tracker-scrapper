package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"tracker-scrapper/internal/features/tracking/domain"
	"tracker-scrapper/internal/features/tracking/ports"
	"tracker-scrapper/internal/features/tracking/service"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTrackingProvider is a mock implementation of TrackingProvider for testing.
type mockTrackingProvider struct {
	supportedCourier string
	returnHistory    *domain.TrackingHistory
	returnError      error
}

// GetTrackingHistory implements TrackingProvider.
func (m *mockTrackingProvider) GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	return m.returnHistory, nil
}

// SupportsCourier implements TrackingProvider.
func (m *mockTrackingProvider) SupportsCourier(courierName string) bool {
	return courierName == m.supportedCourier
}

// TestTrackingHandler_GetTrackingHistory_Success verifies successful tracking retrieval.
func TestTrackingHandler_GetTrackingHistory_Success(t *testing.T) {
	expectedHistory := &domain.TrackingHistory{
		GlobalStatus: domain.TrackingStatusProcessing,
		History:      []domain.TrackingEvent{},
	}

	provider := &mockTrackingProvider{
		supportedCourier: "coordinadora_co",
		returnHistory:    expectedHistory,
	}

	trackingSvc := service.NewTrackingService([]ports.TrackingProvider{provider})
	handler := NewTrackingHandler(trackingSvc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "test-ray-id")
		return c.Next()
	})
	app.Get("/tracking/:number", handler.GetTrackingHistory)

	req := httptest.NewRequest("GET", "/tracking/12345?courier=coordinadora_co", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result domain.TrackingHistory
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, expectedHistory.GlobalStatus, result.GlobalStatus)
}

// TestTrackingHandler_GetTrackingHistory_MissingTrackingNumber verifies tracking number validation.
func TestTrackingHandler_GetTrackingHistory_MissingTrackingNumber(t *testing.T) {
	trackingSvc := service.NewTrackingService([]ports.TrackingProvider{})
	handler := NewTrackingHandler(trackingSvc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "test-ray-id")
		return c.Next()
	})
	app.Get("/tracking/:number", handler.GetTrackingHistory)

	req := httptest.NewRequest("GET", "/tracking/?courier=coordinadora_co", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// TestTrackingHandler_GetTrackingHistory_MissingCourier verifies courier parameter validation.
func TestTrackingHandler_GetTrackingHistory_MissingCourier(t *testing.T) {
	trackingSvc := service.NewTrackingService([]ports.TrackingProvider{})
	handler := NewTrackingHandler(trackingSvc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "test-ray-id")
		return c.Next()
	})
	app.Get("/tracking/:number", handler.GetTrackingHistory)

	req := httptest.NewRequest("GET", "/tracking/12345", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "courier query parameter is required")
	assert.Equal(t, "test-ray-id", errResp.RayID)
}

// TestTrackingHandler_GetTrackingHistory_CourierNotSupported verifies unsupported courier response.
func TestTrackingHandler_GetTrackingHistory_CourierNotSupported(t *testing.T) {
	provider := &mockTrackingProvider{
		supportedCourier: "coordinadora_co",
	}

	trackingSvc := service.NewTrackingService([]ports.TrackingProvider{provider})
	handler := NewTrackingHandler(trackingSvc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "test-ray-id")
		return c.Next()
	})
	app.Get("/tracking/:number", handler.GetTrackingHistory)

	req := httptest.NewRequest("GET", "/tracking/12345?courier=unknown", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "courier not supported")
}
