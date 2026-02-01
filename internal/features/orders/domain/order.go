package domain

import (
	"time"
)

// OrderStatus represents the current state of an order.
type OrderStatus string

const (
	// OrderStatusCreated indicates the order has been placed but not yet shipped.
	OrderStatusCreated OrderStatus = "CREATED"
	// OrderStatusShipped indicates the order has been handed to the carrier.
	OrderStatusShipped OrderStatus = "SHIPPED"
	// OrderStatusCompleted indicates the order has been delivered and finalized.
	OrderStatusCompleted OrderStatus = "COMPLETED"
)

// TrackingInfo represents shipment tracking details for an order.
type TrackingInfo struct {
	// TrackingProvider is the name of the shipping carrier (e.g., DHL, FedEx, USPS).
	TrackingProvider string `json:"tracking_provider"`
	// TrackingNumber is the unique tracking identifier provided by the carrier.
	TrackingNumber string `json:"tracking_number"`
	// DateShipped is the timestamp when the shipment was dispatched.
	DateShipped time.Time `json:"date_shipped,omitempty"`
}

// Order represents a customer order in the system.
type Order struct {
	// ID is the unique identifier for the order.
	ID string `json:"order_id"`
	// Status represents the current state of the order (e.g., CREATED, SHIPPED).
	Status OrderStatus `json:"status"`
	// FirstName is the first name of the customer.
	FirstName string `json:"name"`
	// LastName is the last name of the customer.
	LastName string `json:"last_name"`
	// Address is the shipping address for the order.
	Address string `json:"address"`
	// City is the city of the shipping address.
	City string `json:"city"`
	// State is the state or province of the shipping address.
	State string `json:"state"`
	// Email is the contact email for the customer.
	Email string `json:"email"`
	// Tracking contains shipment tracking information (can be multiple for partial shipments/returns).
	Tracking []TrackingInfo `json:"tracking"`
	// CreatedAt is the timestamp when the order was created.
	CreatedAt time.Time `json:"create_date"`
	// Items contains the list of products included in the order.
	Items []OrderItem `json:"items"`
}

// OrderItem represents an individual item within an order.
type OrderItem struct {
	// Quantity is the number of units purchased.
	Quantity int `json:"quantity"`
	// SKU is the Stock Keeping Unit identifier for the product.
	SKU string `json:"sku"`
	// Name is the descriptive name of the product.
	Name string `json:"name"`
	// Picture is the URL to an image of the product.
	Picture string `json:"picture"`
}
