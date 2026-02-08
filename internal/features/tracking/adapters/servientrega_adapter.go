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
	"github.com/go-rod/rod/lib/proto"
	"go.uber.org/zap"
)

// ServientregaAdapter handles tracking for Servientrega courier.
type ServientregaAdapter struct {
	baseURL string
	logger  *zap.Logger
}

// NewServientregaAdapter creates a new ServientregaAdapter with the given base URL.
func NewServientregaAdapter(baseURL string) *ServientregaAdapter {
	return &ServientregaAdapter{
		baseURL: baseURL,
		logger:  logger.Get(),
	}
}

// servientregaResponse represents the JSON structure returned by the API.
type servientregaResponse struct {
	Results []struct {
		NumeroGuia       string `json:"numeroGuia"`
		EstadoActual     string `json:"estadoActual"`
		FechaEnvio       string `json:"fechaEnvio"` // Format: "31/01/2026 12:51 "
		FechaRealEntrega string `json:"fechaRealEntrega"`
		Movimientos      []struct {
			Fecha      string `json:"fecha"`
			Movimiento string `json:"movimiento"`
			Ubicacion  string `json:"ubicacion"`
			Novedad    string `json:"Novedad"`
			Estado     string `json:"estado"`    // e.g., "Cerrado"
			IdProceso  string `json:"IdProceso"` // Process ID used for Code field
		} `json:"movimientos"`
	} `json:"Results"`
}

// GetTrackingHistory retrieves tracking history from Servientrega.
// GetTrackingHistory retrieves tracking history from Servientrega.
func (a *ServientregaAdapter) GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error) {
	// Create a master context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	a.logger.Info("Starting Servientrega tracking",
		zap.String("tracking_number", trackingNumber),
		zap.Duration("timeout", 60*time.Second),
	)

	// Use baseURL from config (mockable)
	url := fmt.Sprintf("%s%s", a.baseURL, trackingNumber)

	// fast fail: check connectivity first
	if err := a.checkConnectivity(ctx, url); err != nil {
		return nil, fmt.Errorf("connectivity check failed: %w", err)
	}

	a.logger.Debug("Launching browser...")
	// Configure launcher for Docker environment (needs --no-sandbox)
	// Use Context(ctx) to ensure launch respects timeout
	u, err := launcher.New().
		Context(ctx).
		Bin("/usr/bin/chromium").
		Headless(true).
		NoSandbox(true).
		Set("user-agent", stealthUA). // Set User-Agent in browser
		Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	a.logger.Debug("Connecting to browser...")
	browser := rod.New().Context(ctx).ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}
	defer browser.Close()

	a.logger.Debug("Creating page...")
	// Page expects proto.TargetCreateTarget in this version of rod
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	// Measure page operations with the same context
	page = page.Context(ctx)

	// Stealth: Hide webdriver property
	if _, err := page.EvalOnNewDocument("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})"); err != nil {
		a.logger.Warn("Failed to inject stealth script", zap.Error(err))
	}

	a.logger.Debug("Hijacking requests...")
	router := page.HijackRequests()
	defer router.Stop()

	done := make(chan string)

	// Add expects (pattern string, type proto.NetworkResourceType, handler)
	if err := router.Add("*/api/ControlRastreovalidaciones", proto.NetworkResourceTypeXHR, func(ctx *rod.Hijack) {
		a.logger.Debug("Intercepted 'ControlRastreovalidaciones' request")
		if err := ctx.LoadResponse(http.DefaultClient, true); err != nil {
			a.logger.Error("Failed to load response", zap.Error(err))
			return
		}
		body := string(ctx.Response.Body())
		a.logger.Debug("Received response body", zap.Int("length", len(body)))
		done <- body
	}); err != nil {
		return nil, fmt.Errorf("failed to add router handler: %w", err)
	}

	go router.Run()

	// Retry logic for navigation to handle timeouts
	maxRetries := 3
	var navErr error
	for i := 0; i < maxRetries; i++ {
		a.logger.Debug("Navigating to URL",
			zap.String("url", url),
			zap.Int("attempt", i+1),
			zap.Int("max_retries", maxRetries),
		)

		// Use a shorter timeout for individual navigation attempts if needed,
		// but for now relying on the master context is safer to avoid complexity.
		// However, Navigate() might block until load.
		navErr = page.Navigate(url)
		if navErr == nil {
			a.logger.Debug("Navigation successful")
			break
		}
		a.logger.Warn("Navigation failed",
			zap.Error(navErr),
			zap.String("retry_in", "2s"),
		)
		// Wait a bit before retrying, respecting context
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	if navErr != nil {
		return nil, fmt.Errorf("failed to navigate to servientrega after %d attempts: %w", maxRetries, navErr)
	}

	a.logger.Debug("Waiting for API response (intercept)...")
	select {
	case jsonOutput := <-done:
		if strings.TrimSpace(jsonOutput) == "" {
			return nil, fmt.Errorf("empty response from API")
		}
		a.logger.Debug("API response received, parsing...")
		return a.parseResponse([]byte(jsonOutput))
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for API response: %w", ctx.Err())
	}
}

func (a *ServientregaAdapter) parseResponse(body []byte) (*domain.TrackingHistory, error) {
	var resp servientregaResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse servientrega response: %w", err)
	}

	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no tracking results found")
	}

	result := resp.Results[0]

	// Debug: log parsed data
	a.logger.Debug("Parsed tracking data",
		zap.String("numero_guia", result.NumeroGuia),
		zap.String("estado_actual", result.EstadoActual),
		zap.Int("movimientos_count", len(result.Movimientos)),
	)

	// Map Global Status
	status := domain.TrackingStatusProcessing
	switch strings.ToUpper(result.EstadoActual) {
	case "ENTREGADO":
		status = domain.TrackingStatusCompleted
	case "ENTREGADO A REMITENTE":
		status = domain.TrackingStatusReturn // Using 'Return' as mapped in plan, though domain might have different const
	case "EN PROCESAMIENTO":
		status = domain.TrackingStatusProcessing
	default:
		// If unknown, default to Processing or specific unknown logic
		status = domain.TrackingStatusProcessing
	}

	// Map Events
	var events []domain.TrackingEvent
	for _, m := range result.Movimientos {
		location := m.Ubicacion

		// Combine Movimiento and Novedad for description
		description := m.Movimiento
		if m.Novedad != "" {
			description = fmt.Sprintf("%s - %s", description, m.Novedad)
		}

		// Determine type (Mapping "Cerrado" to Checkpoint, otherwise maybe Info?)
		// evtType := "INFO" // Not used in current domain event struct? Code is used.
		// Domain definitions: Text, City, Code, Date.

		// Parse Date
		// Format example: "31/01/2026 12:51 " or "31/01/2026 12:51"
		dateStr := strings.TrimSpace(m.Fecha)
		parsedTime, err := time.Parse("02/01/2006 15:04", dateStr)
		if err != nil {
			// Fallback or log? For now, use zero time or current time?
			// Let's use zero time but log to fmt if possible? No logger here yet.
			parsedTime = time.Time{}
		}

		events = append(events, domain.TrackingEvent{
			Date: parsedTime,
			Text: description,
			City: location,
			Code: m.IdProceso,
		})
	}

	return &domain.TrackingHistory{
		GlobalStatus: status,
		History:      events,
	}, nil
}

// SupportsCourier returns true if this adapter supports servientrega_co.
func (a *ServientregaAdapter) SupportsCourier(courierName string) bool {
	return courierName == "servientrega_co"
}

// stealthUA mimics a real browser to avoid blocking
const stealthUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"

// checkConnectivity performs a simple HTTP request to verify network reachability
func (a *ServientregaAdapter) checkConnectivity(ctx context.Context, urlStr string) error {
	a.logger.Debug("Checking connectivity", zap.String("url", urlStr))

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set stealth User-Agent
	req.Header.Set("User-Agent", stealthUA)

	// Use a shorter timeout for this check (e.g. 10s) derived from the master context
	// But since req uses ctx, it respects the 60s total.
	// Let's rely on standard client.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.logger.Debug("Connectivity check FAILED", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	a.logger.Debug("Connectivity check SUCCESS", zap.String("status", resp.Status))
	return nil
}
