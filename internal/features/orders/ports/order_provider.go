package ports

import "tracker-scrapper/internal/features/orders/domain"

// OrderProvider defines the interface for retrieving external order information.
// This is a Secondary Port (Driven Port).
type OrderProvider interface {
	// GetOrder retrieves an order by its unique identifier (e.g., WooCommerce Order ID).
	GetOrder(orderID string) (*domain.Order, error)
}
