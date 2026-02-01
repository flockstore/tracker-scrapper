package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"tracker-scrapper/internal/core/cache"
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
	// cache is the caching layer for storing tracking results.
	cache cache.Cache
	// cacheTTL is the duration for which tracking data is cached.
	cacheTTL time.Duration
}

// NewTrackingService creates a new TrackingService with cache support.
func NewTrackingService(providers []ports.TrackingProvider, cache cache.Cache, cacheTTL time.Duration) *TrackingService {
	return &TrackingService{
		providers: providers,
		cache:     cache,
		cacheTTL:  cacheTTL,
	}
}

// GetTrackingHistory retrieves tracking history for a given tracking number and courier.
// Uses cache with key format: ts_{courier}_{trackingNumber}
func (s *TrackingService) GetTrackingHistory(trackingNumber, courier string) (*domain.TrackingHistory, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("ts_%s_%s", courier, trackingNumber)

	// Try to get from cache first
	cachedData, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		var history domain.TrackingHistory
		if err := json.Unmarshal(cachedData, &history); err == nil {
			return &history, nil
		}
		// If unmarshal fails, continue to fetch from provider
	}

	// Cache miss or error - fetch from provider
	for _, provider := range s.providers {
		if provider.SupportsCourier(courier) {
			history, err := provider.GetTrackingHistory(trackingNumber)
			if err != nil {
				return nil, fmt.Errorf("failed to get tracking from provider: %w", err)
			}

			// Cache the result
			historyData, err := json.Marshal(history)
			if err == nil {
				// Fire and forget - don't fail if cache write fails
				_ = s.cache.Set(ctx, cacheKey, historyData, s.cacheTTL)
			}

			return history, nil
		}
	}

	return nil, ErrCourierNotSupported
}
