package service

import (
	"errors"
	"fmt"

	"tracker-scrapper/internal/features/tracking/domain"
	"tracker-scrapper/internal/features/tracking/ports"
)

var (
	// ErrCourierNotSupported is returned when no provider supports the requested courier.
	ErrCourierNotSupported = errors.New("courier not supported")
	// ErrTrackingNotFound is returned when the tracking number is not found.
	ErrTrackingNotFound = errors.New("tracking not found")
)

// TrackingService orchestrates tracking requests across multiple courier providers.
type TrackingService struct {
	providers []ports.TrackingProvider
}

// NewTrackingService creates a new TrackingService with the given providers.
func NewTrackingService(providers []ports.TrackingProvider) *TrackingService {
	return &TrackingService{
		providers: providers,
	}
}

// GetTrackingHistory retrieves tracking history for a given tracking number and courier.
func (s *TrackingService) GetTrackingHistory(trackingNumber, courier string) (*domain.TrackingHistory, error) {
	for _, provider := range s.providers {
		if provider.SupportsCourier(courier) {
			history, err := provider.GetTrackingHistory(trackingNumber)
			if err != nil {
				return nil, fmt.Errorf("failed to get tracking from provider: %w", err)
			}
			return history, nil
		}
	}

	return nil, ErrCourierNotSupported
}
