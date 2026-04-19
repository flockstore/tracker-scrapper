package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

// InterrapidisimoAdapter handles tracking for Interrapidisimo courier via scraping.
type InterrapidisimoAdapter struct {
	baseURL string
	proxy   proxy.Settings
	logger  *zap.Logger
}

var interKnownCodes = map[int]bool{
	1:  true, // Recibimos tu envío
	2:  true, // En Centro Logístico Origen / Destino / Tránsito
	3:  true, // Viajando a tu destino
	4:  true, // Viajando a tu destino (variation)
	6:  true, // En camino hacia ti
	7:  true, // No logramos hacer la entrega (Incidence)
	10: true, // Tu envío fue devuelto (Return)
	11: true, // Tu envío fue entregado (Delivered)
	16: true, // Archivada
}

// NewInterrapidisimoAdapter creates a new InterrapidisimoAdapter with the given base URL and proxy settings.
func NewInterrapidisimoAdapter(baseURL string, proxySettings proxy.Settings) *InterrapidisimoAdapter {
	return &InterrapidisimoAdapter{
		baseURL: baseURL,
		proxy:   proxySettings,
		logger:  logger.Get(),
	}
}

// interResponse represents the JSON structure from Interrapidisimo API.
type interResponse struct {
	EstadosGuia []struct {
		EstadoGuia struct {
			IdEstadoGuia          int    `json:"IdEstadoGuia"`
			DescripcionEstadoGuia string `json:"DescripcionEstadoGuia"`
			Ciudad                string `json:"Ciudad"`
			FechaGrabacion        string `json:"FechaGrabacion"`
		} `json:"EstadoGuia"`
	} `json:"EstadosGuia"`
	Guia struct {
		NumeroGuia int64 `json:"NumeroGuia"`
	} `json:"Guia"`
	Success bool   `json:"Success"`
	Message string `json:"Message"`
}

// GetTrackingHistory retrieves tracking history from Interrapidisimo using browser automation.
func (a *InterrapidisimoAdapter) GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error) {
	// Create a master context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start local proxy forwarder if proxy is configured with credentials
	var localProxyAddr string
	var proxyForwarder *proxy.ForwardingProxy
	if a.proxy.HasProxy() && a.proxy.Username != "" && a.proxy.Password != "" {
		var err error
		// Whitelist only Interrapidisimo domains
		proxyForwarder, err = proxy.NewForwardingProxy(a.proxy.FullURL(), "interrapidisimo.com")
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
		localProxyAddr = a.proxy.HostPort()
	}

	a.logger.Debug("Launching browser...",
		zap.Bool("proxy_enabled", a.proxy.HasProxy()),
		zap.String("proxy_addr", localProxyAddr),
	)

	// Configure launcher
	l := launcher.New().
		Context(ctx).
		Headless(true).
		NoSandbox(true)

	// Configure proxy - use local forwarder address (no auth needed)
	if localProxyAddr != "" {
		l = l.Proxy(localProxyAddr)
		a.logger.Debug("Browser configured with proxy", zap.String("proxy", localProxyAddr))
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

	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	page = page.Context(ctx)

	if err := a.navigateWithRetry(page, a.baseURL, 3); err != nil {
		return nil, fmt.Errorf("failed to navigate to interrapidisimo page: %w", err)
	}

	blocked, blockReason := detectInterrapidisimoBlocking(page)
	if blocked {
		return nil, fmt.Errorf("interrapidisimo unavailable or blocked by edge protection: %s", blockReason)
	}

	inputElement, err := a.waitElementWithRetry(page, "#inputGuide", 3, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("tracking input field unavailable: %w", err)
	}

	searchButton, err := a.waitElementWithRetry(page, ".search-button", 3, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("tracking search button unavailable: %w", err)
	}

	// Setup request hijacking
	router := page.HijackRequests()
	defer router.Stop()

	done := make(chan []byte, 1)

	// Intercept the API call
	interceptHandler := func(ctx *rod.Hijack) {
		// Create proxy-aware client if proxy is used
		client := http.DefaultClient
		if localProxyAddr != "" {
			proxyURL, err := url.Parse(localProxyAddr)
			if err != nil {
				a.logger.Error("Failed to parse local proxy URL", zap.Error(err))
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
		select {
		case done <- []byte(ctx.Response.Body()):
		default:
		}
	}

	if err := router.Add("*/ObtenerRastreoGuiasClientePost*", proto.NetworkResourceTypeXHR, interceptHandler); err != nil {
		return nil, fmt.Errorf("failed to add XHR interceptor: %w", err)
	}
	if err := router.Add("*/ObtenerRastreoGuiasClientePost*", proto.NetworkResourceTypeFetch, interceptHandler); err != nil {
		a.logger.Warn("failed to add Fetch interceptor", zap.Error(err))
	}
	go router.Run()

	// Interact with the page
	if err := inputElement.Input(trackingNumber); err != nil {
		return nil, fmt.Errorf("failed to fill tracking number: %w", err)
	}
	if err := searchButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("failed to trigger tracking search: %w", err)
	}

	// Wait for response with timeout
	select {
	case body := <-done:
		// Attempt to unmarshal
		var resp interResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse courier response: %w", err)
		}

		if !resp.Success {
			return nil, fmt.Errorf("courier error: %s", resp.Message)
		}

		return a.mapResponseToDomain(resp)

	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for courier response: %w", ctx.Err())
	}
}

// navigateWithRetry opens a page URL with bounded retries.
func (a *InterrapidisimoAdapter) navigateWithRetry(page *rod.Page, pageURL string, maxRetries int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = page.Navigate(pageURL)
		if lastErr == nil {
			return nil
		}

		a.logger.Warn("Interrapidisimo navigation failed",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(lastErr),
		)
		time.Sleep(2 * time.Second)
	}

	return lastErr
}

// waitElementWithRetry retries selector lookup and visibility checks with backoff.
func (a *InterrapidisimoAdapter) waitElementWithRetry(
	page *rod.Page,
	selector string,
	maxRetries int,
	retryDelay time.Duration,
) (*rod.Element, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		element, err := page.Timeout(15 * time.Second).Element(selector)
		if err == nil {
			err = element.WaitVisible()
			if err == nil {
				return element, nil
			}
		}

		lastErr = err
		a.logger.Warn("Interrapidisimo selector resolution failed",
			zap.String("selector", selector),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(err),
		)
		time.Sleep(retryDelay)
	}

	if lastErr == nil {
		lastErr = errors.New("unknown selector resolution error")
	}

	return nil, lastErr
}

// detectInterrapidisimoBlocking checks for known edge protection pages.
func detectInterrapidisimoBlocking(page *rod.Page) (bool, string) {
	html, err := page.HTML()
	if err != nil {
		return false, ""
	}

	lowered := strings.ToLower(html)
	if strings.Contains(lowered, "errors.edgesuite.net") {
		return true, "errors.edgesuite.net"
	}
	if strings.Contains(lowered, "reference #") && strings.Contains(lowered, "error occurred while processing your request") {
		return true, "akamai reference error"
	}

	return false, ""
}

// mapResponseToDomain converts Interrapidisimo response to domain structure.
func (a *InterrapidisimoAdapter) mapResponseToDomain(resp interResponse) (*domain.TrackingHistory, error) {
	history := &domain.TrackingHistory{
		GlobalStatus: domain.TrackingStatusProcessing, // Default
		History:      make([]domain.TrackingEvent, 0),
	}

	for _, item := range resp.EstadosGuia {
		state := item.EstadoGuia

		// Parse date
		// Format example: "2025-05-10T13:06:23.02" or "2025-04-30T18:53:15.917"
		// We try standard RFC3339-like layouts
		date, _ := time.Parse("2006-01-02T15:04:05", state.FechaGrabacion) // Simplification, might need robust parsing

		event := domain.TrackingEvent{
			Date: date,
			Text: state.DescripcionEstadoGuia,
			City: state.Ciudad,
			Code: strconv.Itoa(state.IdEstadoGuia),
		}
		history.History = append(history.History, event)

		// Determine Global Status based on latest event or specific codes
		// Code 10: RETURN
		// Code 11: DELIVERED
		switch state.IdEstadoGuia {
		case 10:
			history.GlobalStatus = domain.TrackingStatusReturn
		case 11:
			history.GlobalStatus = domain.TrackingStatusCompleted
		case 7:
			history.GlobalStatus = domain.TrackingStatusIncidence
		}

		if !interKnownCodes[state.IdEstadoGuia] {
			a.logger.Warn("Unknown Interrapidisimo status code encountered",
				zap.Int("code", state.IdEstadoGuia),
				zap.String("description", state.DescripcionEstadoGuia),
			)
		}
	}

	return history, nil
}

// SupportsCourier returns true if this adapter supports interrapidisimo_co.
func (a *InterrapidisimoAdapter) SupportsCourier(courierName string) bool {
	return courierName == "interrapidisimo_co"
}
