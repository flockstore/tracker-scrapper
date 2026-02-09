package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/core/proxy"
	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"go.uber.org/zap"
)

// ServientregaAdapter handles tracking for Servientrega courier.
type ServientregaAdapter struct {
	baseURL     string
	proxy       proxy.Settings
	courierName string
	logger      *zap.Logger
}

// NewServientregaAdapter creates a new ServientregaAdapter with the given base URL and proxy settings.
func NewServientregaAdapter(baseURL string, proxySettings proxy.Settings) *ServientregaAdapter {
	return &ServientregaAdapter{
		baseURL:     baseURL,
		proxy:       proxySettings,
		courierName: "servientrega_co",
		logger:      logger.Get(),
	}
}

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
	trackingURL := fmt.Sprintf("%s%s", a.baseURL, trackingNumber)

	// fast fail: check connectivity first
	if err := a.checkConnectivity(ctx, trackingURL); err != nil {
		return nil, fmt.Errorf("connectivity check failed: %w", err)
	}

	// Start local proxy forwarder if proxy is configured with credentials
	// This solves Chromium's limitation of not supporting proxy auth via command line
	var localProxyAddr string
	var proxyForwarder *proxy.ForwardingProxy
	if a.proxy.HasProxy() && a.proxy.Username != "" && a.proxy.Password != "" {
		var err error
		proxyForwarder, err = proxy.NewForwardingProxy(a.proxy.FullURL())
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy forwarder: %w", err)
		}
		localProxyAddr, err = proxyForwarder.Start(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to start proxy forwarder: %w", err)
		}
		defer proxyForwarder.Stop()
		a.logger.Debug("Local proxy forwarder started", zap.String("local_addr", localProxyAddr))
	} else if a.proxy.HasProxy() {
		// Proxy without credentials (IP whitelist mode)
		localProxyAddr = a.proxy.HostPort()
	}

	a.logger.Debug("Launching browser...",
		zap.Bool("proxy_enabled", a.proxy.HasProxy()),
		zap.String("proxy_addr", localProxyAddr),
	)
	// Configure launcher for Docker environment (needs --no-sandbox)
	// Use Context(ctx) to ensure launch respects timeout
	l := launcher.New().
		Context(ctx).
		Bin("/usr/bin/chromium").
		Headless(true).
		NoSandbox(true).
		Set("user-agent", stealthUA) // Set User-Agent in browser

	// Configure proxy - use local forwarder address (no auth needed)
	if localProxyAddr != "" {
		l = l.Proxy(localProxyAddr)
		a.logger.Debug("Browser configured with proxy", zap.String("proxy", localProxyAddr))
	}

	u, err := l.Launch()
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

		// Create proxy-aware client if proxy is used
		client := http.DefaultClient
		if localProxyAddr != "" {
			proxyURL, err := url.Parse(localProxyAddr)
			if err != nil {
				a.logger.Error("Failed to parse local proxy URL", zap.Error(err))
				// continue with default client (might fail if direct access blocked)
			} else {
				client = &http.Client{
					Transport: &http.Transport{
						Proxy: http.ProxyURL(proxyURL),
					},
					Timeout: 30 * time.Second,
				}
			}
		}

		if err := ctx.LoadResponse(client, true); err != nil {
			a.logger.Error("Failed to load response", zap.Error(err))
			return
		}
		done <- ctx.Response.Body()
	}); err != nil {
		return nil, fmt.Errorf("failed to add hijack: %w", err)
	}

	go router.Run()

	// Navigate with retry
	const maxRetries = 3
	var navErr error
	for i := 1; i <= maxRetries; i++ {
		a.logger.Debug("Navigating to URL", zap.String("url", trackingURL), zap.Int("attempt", i), zap.Int("max_retries", maxRetries))
		navErr = page.Navigate(trackingURL)
		if navErr == nil {
			break
		}
		a.logger.Warn("Navigation failed", zap.Error(navErr), zap.Duration("retry_in", 2*time.Second))
		time.Sleep(2 * time.Second)
	}

	// Wait for response
	select {
	case body := <-done:
		a.logger.Debug("Received response from hijacked request")
		var servResp servientregaResponse
		err := json.Unmarshal([]byte(body), &servResp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Servientrega response: %w", err)
		}

		return a.mapResponseToDomain(servResp)

	case <-ctx.Done():
		if navErr != nil {
			// Report navigation error as root cause
			return nil, fmt.Errorf("navigation failed after retries: %w", navErr)
		}
		return nil, fmt.Errorf("timeout waiting for courier response: %w", ctx.Err())
	}
}

// mapResponseToDomain converts servientregaResponse to domain.TrackingHistory.
func (a *ServientregaAdapter) mapResponseToDomain(resp servientregaResponse) (*domain.TrackingHistory, error) {
	history := &domain.TrackingHistory{
		GlobalStatus: domain.TrackingStatusProcessing, // Default
		History:      make([]domain.TrackingEvent, 0),
	}

	// Check for valid response
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results in response (Code: %d)", resp.Code)
	}

	result := resp.Results[0]
	history.GlobalStatus = mapServientregaStatus(result.EstadoActual)

	// Process movements (tracking events)
	// Layout: "31/01/2026 12:51 " (DD/MM/YYYY HH:MM with trailing space)
	const dateLayout = "02/01/2006 15:04"

	for _, mov := range result.Movimientos {
		date, _ := time.Parse(dateLayout, strings.TrimSpace(mov.Fecha))

		event := domain.TrackingEvent{
			Date: date,
			Text: mov.Movimiento,
			City: mov.Ubicacion,
			Code: mov.IdProceso,
		}
		history.History = append(history.History, event)

		// Check if this code is known for analytics purposes
		if !servKnownCodes[mov.IdProceso] {
			a.logger.Warn("Unknown Servientrega movement code encountered",
				zap.String("code", mov.IdProceso),
				zap.String("description", mov.Movimiento),
			)
		}
	}

	return history, nil
}

// SupportsCourier returns true if this adapter supports servientrega_co.
func (a *ServientregaAdapter) SupportsCourier(courierName string) bool {
	return courierName == a.courierName
}

// --- Internal types ---

// servientregaResponse represents the JSON structure from Servientrega API.
type servientregaResponse struct {
	ValidationNumber   int `json:"ValidationNumber"`
	ValidationResponse int `json:"ValidationResponse"`
	Code               int `json:"Code"`
	Results            []struct {
		NumeroGuia   string `json:"numeroGuia"`
		FechaEnvio   string `json:"fechaEnvio"`
		EstadoActual string `json:"estadoActual"`
		Movimientos  []struct {
			Estado     string `json:"estado"`
			Fecha      string `json:"fecha"`
			Movimiento string `json:"movimiento"`
			Ubicacion  string `json:"ubicacion"`
			Novedad    string `json:"Novedad"`
			IdProceso  string `json:"IdProceso"`
		} `json:"movimientos"`
	} `json:"Results"`
}

// Known movement codes for Servientrega
var servKnownCodes = map[string]bool{
	"1":  true, // Guia generada
	"6":  true, // Ingreso al centro logistico
	"12": true, // Salio a ciudad destino
	"15": true, // Llegó a ciudad destino
	"18": true, // En reparto
	"21": true, // Entregado
	"24": true, // Devolución
	"27": true, // Novedad
}

// mapServientregaStatus maps the estado string to our domain status.
func mapServientregaStatus(estado string) domain.TrackingStatus {
	estado = strings.ToUpper(strings.TrimSpace(estado))
	switch {
	case strings.Contains(estado, "ENTREGAD"):
		return domain.TrackingStatusCompleted
	case strings.Contains(estado, "DEVOL") || strings.Contains(estado, "RETURN"):
		return domain.TrackingStatusReturn
	case strings.Contains(estado, "NOVEDAD") || strings.Contains(estado, "INCIDENCIA"):
		return domain.TrackingStatusIncidence
	default:
		return domain.TrackingStatusProcessing
	}
}

// stealthUA mimics a real browser to avoid blocking
const stealthUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"

// checkConnectivity performs a simple HTTP request to verify network reachability
func (a *ServientregaAdapter) checkConnectivity(ctx context.Context, urlStr string) error {
	a.logger.Debug("Checking connectivity",
		zap.String("url", urlStr),
		zap.Bool("proxy_enabled", a.proxy.HasProxy()),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set stealth User-Agent
	req.Header.Set("User-Agent", stealthUA)

	// Create HTTP client with optional proxy
	client := a.getHTTPClient()

	resp, err := client.Do(req)
	if err != nil {
		a.logger.Debug("Connectivity check FAILED", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	a.logger.Debug("Connectivity check SUCCESS", zap.String("status", resp.Status))
	return nil
}

// getHTTPClient returns an HTTP client configured with proxy if enabled.
func (a *ServientregaAdapter) getHTTPClient() *http.Client {
	if !a.proxy.HasProxy() {
		return http.DefaultClient
	}

	proxyURL, err := url.Parse(a.proxy.FullURL())
	if err != nil {
		a.logger.Warn("Invalid proxy URL, using default client", zap.Error(err))
		return http.DefaultClient
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 30 * time.Second,
	}
}
