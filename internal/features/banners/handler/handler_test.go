package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracker-scrapper/internal/features/banners/domain"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBannerService is a mock implementation of ports.BannerService
type MockBannerService struct {
	mock.Mock
}

func (m *MockBannerService) SetBanner(ctx context.Context, title, subtitle string, bannerType domain.BannerType, duration int) error {
	args := m.Called(ctx, title, subtitle, bannerType, duration)
	return args.Error(0)
}

func (m *MockBannerService) GetBanner(ctx context.Context) (*domain.Banner, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Banner), args.Error(1)
}

func (m *MockBannerService) RemoveBanner(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func setupApp(service *MockBannerService) *fiber.App {
	app := fiber.New()
	handler := NewBannerHandler(service)
	app.Post("/banner", handler.SetBanner)
	app.Get("/banner", handler.GetBanner)
	app.Delete("/banner", handler.RemoveBanner)
	return app
}

func TestBannerHandler_SetBanner(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockBannerService)
		app := setupApp(mockService)

		reqBody := CreateBannerRequest{
			Title:    "Test",
			Subtitle: "Subtitle",
			Type:     domain.BannerTypeInfo,
			Duration: 60,
		}
		body, _ := json.Marshal(reqBody)

		mockService.On("SetBanner", mock.Anything, reqBody.Title, reqBody.Subtitle, reqBody.Type, reqBody.Duration).Return(nil).Once()

		req := httptest.NewRequest("POST", "/banner", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("InvalidType", func(t *testing.T) {
		mockService := new(MockBannerService)
		app := setupApp(mockService)

		reqBody := CreateBannerRequest{
			Title: "Test",
			Type:  "INVALID",
		}
		body, _ := json.Marshal(reqBody)

		// The service should return ErrInvalidBannerType
		mockService.On("SetBanner", mock.Anything, reqBody.Title, "", domain.BannerType("INVALID"), 0).Return(domain.ErrInvalidBannerType).Once()

		req := httptest.NewRequest("POST", "/banner", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		if resp.StatusCode != http.StatusBadRequest {
			// Debug
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			t.Logf("Response Body: %s", buf.String())
		}
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		mockService.AssertExpectations(t)
	})
}

func TestBannerHandler_GetBanner(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockBannerService)
		app := setupApp(mockService)

		banner := &domain.Banner{Title: "Test Banner"}
		mockService.On("GetBanner", mock.Anything).Return(banner, nil).Once()

		req := httptest.NewRequest("GET", "/banner", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("NotFound", func(t *testing.T) {
		mockService := new(MockBannerService)
		app := setupApp(mockService)

		mockService.On("GetBanner", mock.Anything).Return(nil, nil).Once()

		req := httptest.NewRequest("GET", "/banner", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		mockService := new(MockBannerService)
		app := setupApp(mockService)

		mockService.On("GetBanner", mock.Anything).Return(nil, errors.New("db error")).Once()

		req := httptest.NewRequest("GET", "/banner", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		if resp.StatusCode != http.StatusInternalServerError {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			t.Logf("Response Body: %s", buf.String())
		}
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockService.AssertExpectations(t)
	})
}

func TestBannerHandler_RemoveBanner(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockBannerService)
		app := setupApp(mockService)

		mockService.On("RemoveBanner", mock.Anything).Return(nil).Once()

		req := httptest.NewRequest("DELETE", "/banner", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockService := new(MockBannerService)
		app := setupApp(mockService)

		mockService.On("RemoveBanner", mock.Anything).Return(errors.New("db error")).Once()

		req := httptest.NewRequest("DELETE", "/banner", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		if resp.StatusCode != http.StatusInternalServerError {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			t.Logf("Response Body: %s", buf.String())
		}
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockService.AssertExpectations(t)
	})
}
