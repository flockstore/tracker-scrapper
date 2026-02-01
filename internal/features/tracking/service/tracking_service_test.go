package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"tracker-scrapper/internal/features/tracking/domain"
	"tracker-scrapper/internal/features/tracking/ports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTrackingProvider is a mock implementation of TrackingProvider for testing.
type mockTrackingProvider struct {
	supportedCourier string
	returnHistory    *domain.TrackingHistory
	returnError      error
}

// GetTrackingHistory implements TrackingProvider.
func (m *mockTrackingProvider) GetTrackingHistory(trackingNumber string) (*domain.TrackingHistory, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	return m.returnHistory, nil
}

// SupportsCourier implements TrackingProvider.
func (m *mockTrackingProvider) SupportsCourier(courierName string) bool {
	return courierName == m.supportedCourier
}

// mockCache is a simple in-memory cache for testing.
type mockCache struct {
	data map[string][]byte
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string][]byte)}
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, errors.New("key not found")
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) Ping(ctx context.Context) error {
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

// TestTrackingService_GetTrackingHistory_Success verifies successful tracking retrieval.
func TestTrackingService_GetTrackingHistory_Success(t *testing.T) {
	expectedHistory := &domain.TrackingHistory{
		GlobalStatus: domain.TrackingStatusProcessing,
		History:      []domain.TrackingEvent{},
	}

	provider := &mockTrackingProvider{
		supportedCourier: "coordinadora_co",
		returnHistory:    expectedHistory,
	}

	mockCache := newMockCache()

	svc := NewTrackingService([]ports.TrackingProvider{provider}, mockCache, 30*time.Second)

	history, err := svc.GetTrackingHistory("12345", "coordinadora_co")

	require.NoError(t, err)
	assert.Equal(t, expectedHistory, history)
}

// TestTrackingService_GetTrackingHistory_CourierNotSupported verifies unsupported courier handling.
func TestTrackingService_GetTrackingHistory_CourierNotSupported(t *testing.T) {
	provider := &mockTrackingProvider{
		supportedCourier: "coordinadora_co",
	}

	mockCache := newMockCache()

	svc := NewTrackingService([]ports.TrackingProvider{provider}, mockCache, 30*time.Second)

	history, err := svc.GetTrackingHistory("12345", "unknown_courier")

	assert.Nil(t, history)
	assert.ErrorIs(t, err, ErrCourierNotSupported)
}

// TestTrackingService_GetTrackingHistory_ProviderError verifies provider error propagation.
func TestTrackingService_GetTrackingHistory_ProviderError(t *testing.T) {
	providerErr := errors.New("provider failure")
	provider := &mockTrackingProvider{
		supportedCourier: "coordinadora_co",
		returnError:      providerErr,
	}

	mockCache := newMockCache()

	svc := NewTrackingService([]ports.TrackingProvider{provider}, mockCache, 30*time.Second)

	history, err := svc.GetTrackingHistory("12345", "coordinadora_co")

	assert.Nil(t, history)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get tracking from provider")
}

// TestTrackingService_GetTrackingHistory_MultipleProviders verifies routing to correct provider.
func TestTrackingService_GetTrackingHistory_MultipleProviders(t *testing.T) {
	provider1 := &mockTrackingProvider{
		supportedCourier: "coordinadora_co",
		returnHistory: &domain.TrackingHistory{
			GlobalStatus: domain.TrackingStatusOrigin,
			History:      []domain.TrackingEvent{},
		},
	}

	provider2 := &mockTrackingProvider{
		supportedCourier: "servientrega_co",
		returnHistory: &domain.TrackingHistory{
			GlobalStatus: domain.TrackingStatusCompleted,
			History:      []domain.TrackingEvent{},
		},
	}

	mockCache := newMockCache()

	svc := NewTrackingService([]ports.TrackingProvider{provider1, provider2}, mockCache, 30*time.Second)

	history, err := svc.GetTrackingHistory("67890", "servientrega_co")

	require.NoError(t, err)
	assert.Equal(t, domain.TrackingStatusCompleted, history.GlobalStatus)
}
