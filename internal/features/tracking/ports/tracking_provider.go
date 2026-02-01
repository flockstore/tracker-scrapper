package ports

import "tracker-scrapper/internal/features/tracking/domain"

// TrackingProvider defines the interface for courier tracking implementations.
type TrackingProvider interface {
	// GetTrackingHistory retrieves the complete tracking history for a given tracking number.
	GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error)
	// SupportsCourier returns true if this provider supports the given courier name.
	SupportsCourier(courierName string) bool
}
