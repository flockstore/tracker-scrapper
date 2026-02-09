package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/core/proxy"
	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
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

	// Open the page
	page := browser.MustPage(a.baseURL)

	// Wait for input field
	page.MustElement("#inputGuide").MustWaitVisible()

	// Setup request hijacking
	router := page.HijackRequests()
	defer router.MustStop()

	done := make(chan []byte)

	// Intercept the API call
	router.MustAdd("*/ObtenerRastreoGuiasClientePost", func(ctx *rod.Hijack) {
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
		done <- []byte(ctx.Response.Body())
	})

	go router.Run()

	// Interact with the page
	page.MustElement("#inputGuide").MustInput(trackingNumber)
	page.MustElement(".search-button").MustClick()

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
