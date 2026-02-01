package handler

import (
	"errors"
	"net/http"

	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/features/orders/service"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// OrderHandler handles HTTP requests related to orders.
type OrderHandler struct {
	// service is the OrderService instance.
	service *service.OrderService
}

// NewOrderHandler creates a new instance of OrderHandler.
func NewOrderHandler(s *service.OrderService) *OrderHandler {
	return &OrderHandler{
		service: s,
	}
}

// GetOrder handles the request to retrieve an order context-aware error handling.
// @Summary Get Order by ID
// @Description Fetch order details using Order ID and Email.
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param email query string true "Customer Email"
// @Success 200 {object} domain.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	email := c.Query("email")

	rayID, ok := c.Locals("requestid").(string)
	if !ok {
		rayID = "unknown"
	}

	if orderID == "" {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Message: "Order ID is required",
			RayID:   rayID,
		})
	}

	if email == "" {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Message: "Email is required",
			RayID:   rayID,
		})
	}

	order, err := h.service.GetOrder(orderID, email)
	if err != nil {
		logger.Get().Error("Failed to fetch order",
			zap.String("order_id", orderID),
			zap.String("ray_id", rayID),
			zap.Error(err),
		)

		status := http.StatusInternalServerError
		msg := "Internal Server Error"

		if errors.Is(err, service.ErrOrderNotFound) {
			status = http.StatusNotFound
			msg = "Order not found"
		} else if errors.Is(err, service.ErrEmailMismatch) {
			status = http.StatusUnauthorized
			msg = "Email mismatch"
		} else {
			msg = err.Error()
		}

		return c.Status(status).JSON(ErrorResponse{
			Message: msg,
			RayID:   rayID,
		})
	}

	return c.Status(http.StatusOK).JSON(order)
}

// ErrorResponse represents the structure of an error response.
type ErrorResponse struct {
	// Message is the error description.
	Message string `json:"message"`
	// RayID is the unique request identifier for debugging.
	RayID string `json:"ray_id"`
}
