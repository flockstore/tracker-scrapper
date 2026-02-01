package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tracker-scrapper/internal/core/cache"
	"tracker-scrapper/internal/features/orders/domain"
	"tracker-scrapper/internal/features/orders/ports"
)

// ErrOrderNotFound is returned when the order does not exist.
var ErrOrderNotFound = errors.New("order not found")

// ErrEmailMismatch is returned when the provided email does not match the order's email.
var ErrEmailMismatch = errors.New("email does not match order record")

// OrderService handles the business logic for retrieving and validating orders.
type OrderService struct {
	// provider is the interface for fetching order data from external sources.
	provider ports.OrderProvider
	// cache is the caching layer for storing retrieved orders.
	cache cache.Cache
	// cacheTTL is the duration for which orders are cached.
	cacheTTL time.Duration
}

// NewOrderService creates a new instance of OrderService with cache support.
func NewOrderService(provider ports.OrderProvider, cache cache.Cache, cacheTTL time.Duration) *OrderService {
	return &OrderService{
		provider: provider,
		cache:    cache,
		cacheTTL: cacheTTL,
	}
}

// GetOrder retrieves an order by ID and validates that the provided email matches the order's email.
// Uses cache with key format: order_{orderID}_{email}
func (s *OrderService) GetOrder(orderID, email string) (*domain.Order, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("order_%s_%s", orderID, email)

	// Try to get from cache first
	cachedData, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		var order domain.Order
		if err := json.Unmarshal(cachedData, &order); err == nil {
			return &order, nil
		}
		// If unmarshal fails, continue to fetch from provider
	}

	// Cache miss or error - fetch from provider
	order, err := s.provider.GetOrder(orderID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, ErrOrderNotFound
	}

	// Validate email before caching
	if !strings.EqualFold(order.Email, email) {
		return nil, ErrEmailMismatch
	}

	// Cache the validated order
	orderData, err := json.Marshal(order)
	if err == nil {
		// Fire and forget - don't fail if cache write fails
		_ = s.cache.Set(ctx, cacheKey, orderData, s.cacheTTL)
	}

	return order, nil
}
