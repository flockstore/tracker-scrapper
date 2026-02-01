package adapter

import (
	"tracker-scrapper/internal/features/tracking/domain"
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

// GetTrackingHistory retrieves tracking history from Servientrega (stub implementation).
func (a *ServientregaAdapter) GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error) {
	return &domain.TrackingHistory{
		GlobalStatus: domain.TrackingStatusProcessing,
		History:      []domain.TrackingEvent{},
	}, nil
}

// SupportsCourier returns true if this adapter supports servientrega_co.
func (a *ServientregaAdapter) SupportsCourier(courierName string) bool {
	return courierName == "servientrega_co"
}
