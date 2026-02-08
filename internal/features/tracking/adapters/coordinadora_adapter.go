package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"go.uber.org/zap"
)

// CoordinadoraAdapter handles tracking for Coordinadora courier via scraping.
type CoordinadoraAdapter struct {
	baseURL string
	proxy   ProxySettings
	logger  *zap.Logger
}

var coordKnownCodes = map[string]bool{
	"2": true, // EN TERMINAL ORIGEN
	"3": true, // EN TRANSPORTE
	"4": true, // EN TERMINAL DESTINO
	"5": true, // EN REPARTO
	"6": true, // ENTREGADA
	"8": true, // CERRADO POR INCIDENCIA / RETURN
	// Incidence variations (7xx)
	"700":    true, // Incidence
	"701":    true, // Visita no entrega
	"701_4":  true, // Novedad tiene solución
	"701_10": true, // Novedad tiene solución
	"728":    true, // Destinatario no cancela
	"733":    true, // Afectacion tiempo entrega
	// Other
	"post_binded": true, // Nueva guia generada
}

// NewCoordinadoraAdapter creates a new CoordinadoraAdapter with the given base URL and proxy settings.
func NewCoordinadoraAdapter(baseURL string, proxy ProxySettings) *CoordinadoraAdapter {
	return &CoordinadoraAdapter{
		baseURL: baseURL,
		proxy:   proxy,
		logger:  logger.Get(),
	}
}

// coordinadoraResponse represents the JSON structure from Coordinadora API.
type coordinadoraResponse struct {
	TrackingNumber string `json:"tracking_number"`
	History        []struct {
		Code        string `json:"code"`
		Date        string `json:"date"`
		Description string `json:"description"`
	} `json:"history"`
}

// GetTrackingHistory retrieves tracking history from Coordinadora using browser automation.
func (a *CoordinadoraAdapter) GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error) {
	// Create a master context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pageURL := fmt.Sprintf(a.baseURL, trackingNumber)
	if !strings.Contains(a.baseURL, "%s") {
		// Fallback: Check if it ends with =, implying a query param is ready
		if strings.HasSuffix(a.baseURL, "=") {
			pageURL = a.baseURL + trackingNumber
		} else {
			pageURL = fmt.Sprintf("%s?guia=%s", a.baseURL, trackingNumber)
		}
	}

	a.logger.Debug("Launching browser...",
		zap.Bool("proxy_enabled", a.proxy.HasProxy()),
		zap.String("proxy_host", a.proxy.HostPort()),
	)

	// Configure launcher
	l := launcher.New().
		Context(ctx).
		Headless(true).
		NoSandbox(true)

	// Configure proxy if enabled
	if a.proxy.HasProxy() {
		l = l.Proxy(a.proxy.HostPort())
		a.logger.Debug("Browser configured with proxy")
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().Context(ctx).ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}
	defer browser.Close()

	// Handle proxy authentication if credentials were provided
	if a.proxy.HasProxy() && a.proxy.Username != "" && a.proxy.Password != "" {
		go browser.MustHandleAuth(a.proxy.Username, a.proxy.Password)()
		a.logger.Debug("Proxy authentication configured")
	}

	page := browser.MustPage(pageURL)

	router := page.HijackRequests()
	defer router.MustStop()

	done := make(chan []byte)

	// Pattern from user example: */wp-json/rgc/v1/detail_tracking*
	router.MustAdd("*/wp-json/rgc/v1/detail_tracking*", func(ctx *rod.Hijack) {
		if err := ctx.LoadResponse(http.DefaultClient, true); err != nil {
			return
		}
		done <- []byte(ctx.Response.Body())
	})

	go router.Run()

	// Wait for response
	select {
	case body := <-done:
		var resp coordinadoraResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse courier response: %w", err)
		}
		return a.mapResponseToDomain(resp)

	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for courier response: %w", ctx.Err())
	}
}

// mapResponseToDomain converts Coordinadora response to domain structure.
func (a *CoordinadoraAdapter) mapResponseToDomain(resp coordinadoraResponse) (*domain.TrackingHistory, error) {
	history := &domain.TrackingHistory{
		GlobalStatus: domain.TrackingStatusProcessing, // Default
		History:      make([]domain.TrackingEvent, 0),
	}

	// Layout: "2023-12-28 10:50:44"
	const dateLayout = "2006-01-02 15:04:05"

	for _, item := range resp.History {
		date, _ := time.Parse(dateLayout, item.Date)

		event := domain.TrackingEvent{
			Date: date,
			Text: item.Description,
			City: "", // Coordinadora history items don't strictly have city
			Code: item.Code,
		}
		history.History = append(history.History, event)

		// Status Mapping Logic
		// 6 -> COMPLETED
		// 8 -> RETURN
		// 7... -> INCIDENCE
		switch {
		case item.Code == "6":
			history.GlobalStatus = domain.TrackingStatusCompleted
		case item.Code == "8":
			history.GlobalStatus = domain.TrackingStatusReturn
		case strings.HasPrefix(item.Code, "7"):
			history.GlobalStatus = domain.TrackingStatusIncidence
		}

		// Check known codes.
		// For Coordinadora, 7xx codes are virtually infinite variations of incidence.
		// We treat any prefix "7" as known incidence category.
		isKnown := coordKnownCodes[item.Code] || strings.HasPrefix(item.Code, "7")
		if !isKnown {
			a.logger.Warn("Unknown Coordinadora status code encountered",
				zap.String("code", item.Code),
				zap.String("description", item.Description),
			)
		}
	}

	return history, nil
}

// SupportsCourier returns true if this adapter supports coordinadora_co.
func (a *CoordinadoraAdapter) SupportsCourier(courierName string) bool {
	return courierName == "coordinadora_co"
}
