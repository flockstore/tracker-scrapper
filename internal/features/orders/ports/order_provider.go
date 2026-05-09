package ports

import (
	"errors"

	"tracker-scrapper/internal/features/orders/domain"
)

// ErrOrderNotFound is returned when the upstream order lookup has no match.
var ErrOrderNotFound = errors.New("order not found")

// OrderProvider defines the interface for retrieving external order information.
// This is a Secondary Port (Driven Port).
type OrderProvider interface {
	// GetOrder retrieves an order by its identifier and customer email.
	GetOrder(orderID string, email string) (*domain.Order, error)
}
