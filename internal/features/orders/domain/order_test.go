package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOrder_MarshalJSON(t *testing.T) {
	now := time.Now()
	order := Order{
		ID:             "123",
		Status:         OrderStatusCreated,
		FirstName:      "John",
		LastName:       "Doe",
		Address:        "123 Main St",
		City:           "New York",
		State:          "NY",
		Email:          "john@example.com",
		TrackingNumber: "TRACK123",
		Carrier:        "DHL",
		CreatedAt:      now,
		Items: []OrderItem{
			{
				Quantity: 1,
				SKU:      "SKU-1",
				Name:     "Item 1",
				Picture:  "http://example.com/pic.jpg",
			},
		},
	}

	data, err := json.Marshal(order)
	assert.NoError(t, err)

	// Verify key existence in JSON
	jsonString := string(data)
	assert.Contains(t, jsonString, `"order_id":"123"`)
	assert.Contains(t, jsonString, `"status":"CREATED"`)
	assert.Contains(t, jsonString, `"name":"John"`)
	assert.Contains(t, jsonString, `"items":[{`)
}

func TestOrderStatus_Values(t *testing.T) {
	assert.Equal(t, OrderStatus("CREATED"), OrderStatusCreated)
	assert.Equal(t, OrderStatus("SHIPPED"), OrderStatusShipped)
	assert.Equal(t, OrderStatus("COMPLETED"), OrderStatusCompleted)
}
