package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ServientregaAdapter handles tracking for Servientrega courier.
type ServientregaAdapter struct {
	baseURL string
}

// NewServientregaAdapter creates a new ServientregaAdapter with the given base URL.
func NewServientregaAdapter(baseURL string) *ServientregaAdapter {
	return &ServientregaAdapter{
		baseURL: baseURL,
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
	// Use baseURL from config (mockable)
	url := fmt.Sprintf("%s%s", a.baseURL, trackingNumber)

	// Configure launcher for Docker environment (needs --no-sandbox)
	u, err := launcher.New().
		Headless(true).
		NoSandbox(true).
		Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}
	defer browser.Close()

	// Page expects proto.TargetCreateTarget in this version of rod
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	router := page.HijackRequests()
	defer router.Stop()

	done := make(chan string)

	// Add expects (pattern string, type proto.NetworkResourceType, handler)
	// NetworkResourceTypeUndefined is likely what we want if we don't care, but let's check if it exists in this version.
	// If undefined is not found, we can try leaving it out if the signature allows, but the error said it wants it.
	// Let's try proto.NetworkResourceTypeScripts as a fallback or just empty if it's an enum.
	// Error was: undefined: proto.NetworkResourceTypeUndefined.
	// In v0.116.2 it might be proto.NetworkResourceType("")? Or valid types are e.g. proto.NetworkResourceTypeXHR.
	// Since we are hijacking "*/api/ControlRastreovalidaciones", it's likely an XHR/Fetch.
	if err := router.Add("*/api/ControlRastreovalidaciones", proto.NetworkResourceTypeXHR, func(ctx *rod.Hijack) {
		if err := ctx.LoadResponse(http.DefaultClient, true); err != nil {
			return
		}
		done <- string(ctx.Response.Body())
	}); err != nil {
		return nil, fmt.Errorf("failed to add router handler: %w", err)
	}

	go router.Run()

	// Retry logic for navigation to handle timeouts
	maxRetries := 3
	var navErr error
	for i := 0; i < maxRetries; i++ {
		navErr = page.Navigate(url)
		if navErr == nil {
			break
		}
		// Wait a bit before retrying
		time.Sleep(2 * time.Second)
	}

	if navErr != nil {
		return nil, fmt.Errorf("failed to navigate to servientrega after %d attempts: %w", maxRetries, navErr)
	}

	select {
	case jsonOutput := <-done:
		if strings.TrimSpace(jsonOutput) == "" {
			return nil, fmt.Errorf("empty response from API")
		}
		return a.parseResponse([]byte(jsonOutput))
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for API response")
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
	fmt.Printf("DEBUG: NumeroGuia=%s, EstadoActual=%s, Movimientos count=%d\n",
		result.NumeroGuia, result.EstadoActual, len(result.Movimientos))

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
