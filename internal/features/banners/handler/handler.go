package handler

import (
	"net/http"

	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/features/banners/domain"
	"tracker-scrapper/internal/features/banners/ports"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// BannerHandler handles HTTP requests for banners.
type BannerHandler struct {
	service ports.BannerService
}

// NewBannerHandler creates a new BannerHandler.
func NewBannerHandler(service ports.BannerService) *BannerHandler {
	return &BannerHandler{
		service: service,
	}
}

// CreateBannerRequest represents the request body for creating a banner.
type CreateBannerRequest struct {
	Title    string            `json:"title"`
	Subtitle string            `json:"subtitle"`
	Type     domain.BannerType `json:"type"`
	Duration int               `json:"duration"` // Seconds
}

// SetBanner handles POST /banner.
// @Summary Set a new banner
// @Description Creates or updates the site-wide banner alert.
// @Tags Banner
// @Accept json
// @Produce json
// @Param banner body CreateBannerRequest true "Banner details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /banner [post]
func (h *BannerHandler) SetBanner(c *fiber.Ctx) error {
	var req CreateBannerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	ctx := c.Context()
	if err := h.service.SetBanner(ctx, req.Title, req.Subtitle, req.Type, req.Duration); err != nil {
		if err == domain.ErrInvalidBannerType {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid banner type. Must be INFO, WARNING, or DANGER",
			})
		}
		logger.Get().Error("Failed to set banner", zap.Error(err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Banner set successfully",
	})
}

// GetBanner handles GET /banner.
// @Summary Get the current banner
// @Description Retrieves the active site-wide banner alert.
// @Tags Banner
// @Produce json
// @Success 200 {object} domain.Banner
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /banner [get]
func (h *BannerHandler) GetBanner(c *fiber.Ctx) error {
	ctx := c.Context()
	banner, err := h.service.GetBanner(ctx)
	if err != nil {
		logger.Get().Error("Failed to get banner", zap.Error(err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	if banner == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "No active banner",
		})
	}

	return c.Status(http.StatusOK).JSON(banner)
}

// RemoveBanner handles DELETE /banner.
// @Summary Remove the current banner
// @Description Manually removes the active site-wide banner alert.
// @Tags Banner
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /banner [delete]
func (h *BannerHandler) RemoveBanner(c *fiber.Ctx) error {
	ctx := c.Context()
	if err := h.service.RemoveBanner(ctx); err != nil {
		logger.Get().Error("Failed to remove banner", zap.Error(err))
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Banner removed successfully",
	})
}
