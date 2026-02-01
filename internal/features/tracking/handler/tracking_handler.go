package handler

import (
	"tracker-scrapper/internal/features/tracking/service"

	"github.com/gofiber/fiber/v2"
)

// TrackingHandler handles HTTP requests for tracking operations.
type TrackingHandler struct {
	trackingService *service.TrackingService
}

// NewTrackingHandler creates a new TrackingHandler.
func NewTrackingHandler(trackingService *service.TrackingService) *TrackingHandler {
	return &TrackingHandler{
		trackingService: trackingService,
	}
}

// ErrorResponse represents an error response with Ray ID.
type ErrorResponse struct {
	// Message is the error description.
	Message string `json:"message"`
	// RayID is the unique request identifier for tracing.
	RayID string `json:"ray_id,omitempty"`
}

// GetTrackingHistory godoc
// @Summary Get tracking history for a shipment
// @Description Retrieves the complete tracking history for a given tracking number and courier
// @Tags tracking
// @Accept json
// @Produce json
// @Param number path string true "Tracking Number"
// @Param courier query string true "Courier name (e.g., coordinadora_co, servientrega_co)"
// @Success 200 {object} domain.TrackingHistory
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /tracking/{number} [get]
func (h *TrackingHandler) GetTrackingHistory(c *fiber.Ctx) error {
	trackingNumber := c.Params("number")
	if trackingNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "tracking number is required",
			RayID:   c.Locals("requestid").(string),
		})
	}

	courier := c.Query("courier")
	if courier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "courier query parameter is required",
			RayID:   c.Locals("requestid").(string),
		})
	}

	history, err := h.trackingService.GetTrackingHistory(trackingNumber, courier)
	if err != nil {
		if err == service.ErrCourierNotSupported {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Message: "courier not supported",
				RayID:   c.Locals("requestid").(string),
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: err.Error(),
			RayID:   c.Locals("requestid").(string),
		})
	}

	return c.JSON(history)
}
