package service

import (
	"errors"
	"strings"

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
}

// NewOrderService creates a new instance of OrderService.
func NewOrderService(provider ports.OrderProvider) *OrderService {
	return &OrderService{
		provider: provider,
	}
}

// GetOrder retrieves an order by ID and validates that the provided email matches the order's email.
func (s *OrderService) GetOrder(orderID, email string) (*domain.Order, error) {
	order, err := s.provider.GetOrder(orderID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, ErrOrderNotFound
	}

	if !strings.EqualFold(order.Email, email) {
		return nil, ErrEmailMismatch
	}

	return order, nil
}
