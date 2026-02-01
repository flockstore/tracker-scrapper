package adapter

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/features/orders/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWooCommerceAdapter_GetOrder_Success verifies successful order fetching and mapping.
func TestWooCommerceAdapter_GetOrder_Success(t *testing.T) {
	mockResponse := `{
		"id": 123,
		"status": "processing",
		"date_created": "2023-10-25T10:00:00",
		"billing": {
			"first_name": "John",
			"last_name": "Doe",
			"email": "john.doe@example.com"
		},
		"shipping": {
			"address_1": "123 Main St",
			"city": "Test City",
			"state": "TS"
		},
		"line_items": [
			{
				"id": 1,
				"name": "Product A",
				"sku": "SKU-A",
				"quantity": 2,
				"image": {
					"src": "http://example.com/a.jpg"
				}
			}
		],
		"fee_lines": [],
		"shipping_lines": [],
		"meta_data": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/wp-json/wc/v3/orders/123", r.URL.Path)

		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("ck_test:cs_test"))
		assert.Equal(t, expectedAuth, r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	cfg := config.WooCommerceConfig{
		URL:            server.URL,
		ConsumerKey:    "ck_test",
		ConsumerSecret: "cs_test",
	}

	adapter := NewWooCommerceAdapter(cfg)
	order, err := adapter.GetOrder("123")

	require.NoError(t, err)
	require.NotNil(t, order)

	assert.Equal(t, "123", order.ID)
	assert.Equal(t, domain.OrderStatusCreated, order.Status)
	assert.Equal(t, "John", order.FirstName)
	assert.Equal(t, "Doe", order.LastName)
	assert.Equal(t, "123 Main St", order.Address)
	assert.Equal(t, "Test City", order.City)
	assert.Equal(t, "TS", order.State)
	assert.Equal(t, "john.doe@example.com", order.Email)
	assert.Empty(t, order.Tracking)

	require.Len(t, order.Items, 1)
	assert.Equal(t, "Product A", order.Items[0].Name)
	assert.Equal(t, "SKU-A", order.Items[0].SKU)
	assert.Equal(t, 2, order.Items[0].Quantity)
	assert.Equal(t, "http://example.com/a.jpg", order.Items[0].Picture)

	expectedDate, _ := time.Parse("2006-01-02T15:04:05", "2023-10-25T10:00:00")
	assert.True(t, expectedDate.Equal(order.CreatedAt), "Date should match")
}

// TestWooCommerceAdapter_GetOrder_WithShippingLineTracking verifies tracking from shipping_lines metadata.
func TestWooCommerceAdapter_GetOrder_WithShippingLineTracking(t *testing.T) {
	mockResponse := `{
		"id": 456,
		"status": "completed",
		"date_created": "2023-10-26T12:00:00",
		"billing": {
			"first_name": "Jane",
			"last_name": "Smith",
			"email": "jane@example.com"
		},
		"shipping": {
			"address_1": "456 Elm St",
			"city": "Sample City",
			"state": "SC"
		},
		"line_items": [],
		"fee_lines": [],
		"shipping_lines": [
			{
				"method_id": "skydropx",
				"method_title": "coordinadora_co",
				"meta_data": [
					{
						"key": "Tracking Number",
						"value": "93202303516"
					},
					{
						"key": "Tracking Company",
						"value": "coordinadora_co"
					}
				]
			}
		],
		"meta_data": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	adapter := NewWooCommerceAdapter(config.WooCommerceConfig{URL: server.URL})
	order, err := adapter.GetOrder("456")

	require.NoError(t, err)
	require.NotNil(t, order)

	assert.Equal(t, domain.OrderStatusShipped, order.Status)
	require.Len(t, order.Tracking, 1)

	assert.Equal(t, "coordinadora_co", order.Tracking[0].TrackingProvider)
	assert.Equal(t, "93202303516", order.Tracking[0].TrackingNumber)
}

// TestWooCommerceAdapter_GetOrder_WithFeeLines verifies fee_lines are included as items.
func TestWooCommerceAdapter_GetOrder_WithFeeLines(t *testing.T) {
	mockResponse := `{
		"id": 789,
		"status": "processing",
		"date_created": "2023-10-27T15:00:00",
		"billing": {"first_name": "Bob", "last_name": "Brown", "email": "bob@example.com"},
		"shipping": {"address_1": "789 Oak St", "city": "Town", "state": "TN"},
		"line_items": [
			{
				"name": "Regular Product",
				"sku": "REG-SKU",
				"quantity": 1,
				"image": {"src": "http://example.com/product.jpg"}
			}
		],
		"fee_lines": [
			{"name": "Journey Camo Blanco"}
		],
		"shipping_lines": [],
		"meta_data": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	adapter := NewWooCommerceAdapter(config.WooCommerceConfig{URL: server.URL})
	order, err := adapter.GetOrder("789")

	require.NoError(t, err)
	require.Len(t, order.Items, 2)

	assert.Equal(t, "Regular Product", order.Items[0].Name)
	assert.Equal(t, "REG-SKU", order.Items[0].SKU)

	assert.Equal(t, "Journey Camo Blanco", order.Items[1].Name)
	assert.Equal(t, "", order.Items[1].SKU)
	assert.Equal(t, "", order.Items[1].Picture)
}

// TestWooCommerceAdapter_GetOrder_LegacyTracking verifies fallback to legacy metadata.
func TestWooCommerceAdapter_GetOrder_LegacyTracking(t *testing.T) {
	mockResponse := `{
		"id": 890,
		"status": "processing",
		"date_created": "2023-10-28T10:00:00",
		"billing": {"first_name": "Alice", "last_name": "Green", "email": "alice@example.com"},
		"shipping": {"address_1": "890 Pine St", "city": "Village", "state": "VG"},
		"line_items": [],
		"fee_lines": [],
		"shipping_lines": [],
		"meta_data": [
			{"key": "_tracking_number", "value": "LEGACY456"},
			{"key": "_tracking_company", "value": "DHL"}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	adapter := NewWooCommerceAdapter(config.WooCommerceConfig{URL: server.URL})
	order, err := adapter.GetOrder("890")

	require.NoError(t, err)
	require.Len(t, order.Tracking, 1)

	assert.Equal(t, "DHL", order.Tracking[0].TrackingProvider)
	assert.Equal(t, "LEGACY456", order.Tracking[0].TrackingNumber)
	assert.Equal(t, domain.OrderStatusShipped, order.Status)
}

// TestWooCommerceAdapter_GetOrder_NotFound verifies 404 handling.
func TestWooCommerceAdapter_GetOrder_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := config.WooCommerceConfig{
		URL: server.URL,
	}
	adapter := NewWooCommerceAdapter(cfg)

	order, err := adapter.GetOrder("999")
	require.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "order not found")
}

// TestWooCommerceAdapter_GetOrder_MappedStatus tests the status mapping logic.
func TestWooCommerceAdapter_GetOrder_MappedStatus(t *testing.T) {
	tests := []struct {
		wcStatus     string
		hasTracking  bool
		domainStatus domain.OrderStatus
	}{
		{"pending", false, domain.OrderStatusCreated},
		{"processing", false, domain.OrderStatusCreated},
		{"completed", false, domain.OrderStatusShipped},
		{"processing", true, domain.OrderStatusShipped},
		{"unknown", false, domain.OrderStatusCreated},
	}

	for _, tt := range tests {
		name := tt.wcStatus
		if tt.hasTracking {
			name += "_with_tracking"
		}
		t.Run(name, func(t *testing.T) {
			var tracking []domain.TrackingInfo
			if tt.hasTracking {
				tracking = []domain.TrackingInfo{{TrackingProvider: "DHL", TrackingNumber: "123"}}
			}
			res := mapStatus(tt.wcStatus, tracking)
			assert.Equal(t, tt.domainStatus, res)
		})
	}
}

// TestWooCommerceAdapter_HealthCheck tests the HealthCheck logic.
func TestWooCommerceAdapter_HealthCheck(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/wp-json/wc/v3/orders", r.URL.Path)
			assert.Equal(t, "1", r.URL.Query().Get("per_page"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := config.WooCommerceConfig{URL: server.URL}
		adapter := NewWooCommerceAdapter(cfg)

		err := adapter.HealthCheck()
		assert.NoError(t, err)
	})

	t.Run("Failure_500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cfg := config.WooCommerceConfig{URL: server.URL}
		adapter := NewWooCommerceAdapter(cfg)

		err := adapter.HealthCheck()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status: 500")
	})

	t.Run("Failure_Network", func(t *testing.T) {
		cfg := config.WooCommerceConfig{URL: "http://invalid-url.local"}
		adapter := NewWooCommerceAdapter(cfg)
		err := adapter.HealthCheck()
		assert.Error(t, err)
	})
}
