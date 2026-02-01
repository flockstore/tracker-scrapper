package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/go-rod/rod"
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
func (a *ServientregaAdapter) GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error) {
	// Use exact same pattern as working standalone scraper
	url := fmt.Sprintf("https://mobile.servientrega.com/WebSitePortal/RastreoEnvioDetalle.html?Guia=%s", trackingNumber)

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

	router := page.HijackRequests()
	defer router.MustStop()

	done := make(chan string)

	router.MustAdd("*/api/ControlRastreovalidaciones", func(ctx *rod.Hijack) {
		if err := ctx.LoadResponse(http.DefaultClient, true); err != nil {
			return
		}
		done <- string(ctx.Response.Body())
	})

	go router.Run()

	page.MustNavigate(url)

	jsonOutput := <-done

	if strings.TrimSpace(jsonOutput) == "" {
		return nil, fmt.Errorf("empty response from API")
	}

	return a.parseResponse([]byte(jsonOutput))
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
