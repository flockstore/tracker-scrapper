package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/features/orders/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMannaiahAdapter_GetOrder_Success verifies successful Mannaiah lookup and mapping.
func TestMannaiahAdapter_GetOrder_Success(t *testing.T) {
	tokenRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/token" {
			tokenRequests++
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
			assert.Equal(t, "m2m-client", r.Form.Get("client_id"))
			assert.Equal(t, "m2m-secret", r.Form.Get("client_secret"))
			assert.Equal(t, "order:view contact:view product:view shipping:quotations", r.Form.Get("scope"))
			writeJSON(t, w, map[string]interface{}{
				"access_token": "token-123",
				"expires_in":   3600,
			})
			return
		}

		assert.Equal(t, "Bearer token-123", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/contacts":
			assert.Equal(t, "john@example.com", r.URL.Query().Get("email"))
			writeJSON(t, w, map[string]interface{}{
				"data": []map[string]interface{}{{
					"id":        "contact-1",
					"email":     "john@example.com",
					"firstName": "John",
					"lastName":  "Doe",
					"cityCode":  "11001",
				}},
			})
		case "/orders":
			assert.Equal(t, "1025395", r.URL.Query().Get("identifier"))
			assert.Equal(t, "contact-1", r.URL.Query().Get("contactId"))
			writeJSON(t, w, map[string]interface{}{
				"data": []map[string]interface{}{{
					"id":            "order-internal-1",
					"identifier":    "1025395",
					"currentStatus": "CREATED",
					"paymentMethod": "Wompi",
					"createdAt":     "2026-05-08T05:15:00Z",
					"shippingAddress": map[string]interface{}{
						"address":  "123 Main St",
						"cityCode": "11001",
					},
					"items": []map[string]interface{}{{
						"quantity":      2,
						"sku":           "SKU-A",
						"alternateName": "Product A",
					}},
				}},
			})
		case "/shipping/marks":
			assert.Equal(t, "order-internal-1", r.URL.Query().Get("orderID"))
			writeJSON(t, w, map[string]interface{}{
				"data": []map[string]interface{}{{
					"carrierId":      "coordinadora_co",
					"trackingNumber": "TRACK-1",
				}},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewMannaiahAdapter(config.MannaiahConfig{
		BackendURL:    server.URL,
		TokenEndpoint: server.URL + "/oidc/token",
		AppID:         "m2m-client",
		AppSecret:     "m2m-secret",
		Scope:         "order:view contact:view product:view shipping:quotations",
	})

	order, err := adapter.GetOrder("1025395", "john@example.com")

	require.NoError(t, err)
	require.NotNil(t, order)
	assert.Equal(t, "1025395", order.ID)
	assert.Equal(t, domain.OrderStatusShipped, order.Status)
	assert.Equal(t, "John", order.FirstName)
	assert.Equal(t, "Doe", order.LastName)
	assert.Equal(t, "123 Main St", order.Address)
	assert.Equal(t, "11001", order.City)
	assert.Equal(t, "john@example.com", order.Email)
	assert.Equal(t, "Wompi", order.PaymentMethod)
	require.Len(t, order.Tracking, 1)
	assert.Equal(t, "TRACK-1", order.Tracking[0].TrackingNumber)
	require.Len(t, order.Items, 1)
	assert.Equal(t, "Product A", order.Items[0].Name)
	assert.Equal(t, "SKU-A", order.Items[0].SKU)
	assert.Equal(t, 2, order.Items[0].Quantity)
	assert.Equal(t, 1, tokenRequests)
}

// TestMannaiahAdapter_GetOrder_NotFound verifies missing contacts become not-found.
func TestMannaiahAdapter_GetOrder_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/token" {
			writeJSON(t, w, map[string]interface{}{
				"access_token": "token-123",
				"expires_in":   3600,
			})
			return
		}
		writeJSON(t, w, map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	adapter := NewMannaiahAdapter(config.MannaiahConfig{
		BackendURL:    server.URL,
		TokenEndpoint: server.URL + "/oidc/token",
		AppID:         "m2m-client",
		AppSecret:     "m2m-secret",
		Scope:         "order:view contact:view product:view shipping:quotations",
	})

	order, err := adapter.GetOrder("1025395", "missing@example.com")

	require.Error(t, err)
	assert.Nil(t, order)
}

// TestMannaiahAdapter_GetOrder_ForbiddenLogsStatus verifies upstream 403 is preserved.
func TestMannaiahAdapter_GetOrder_ForbiddenLogsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/token" {
			writeJSON(t, w, map[string]interface{}{
				"access_token": "token-123",
				"expires_in":   3600,
			})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Forbidden resource"}`))
	}))
	defer server.Close()

	adapter := NewMannaiahAdapter(config.MannaiahConfig{
		BackendURL:    server.URL,
		TokenEndpoint: server.URL + "/oidc/token",
		AppID:         "m2m-client",
		AppSecret:     "m2m-secret",
		Scope:         "order:view contact:view product:view shipping:quotations",
	})

	order, err := adapter.GetOrder("1025395", "john@example.com")

	require.Error(t, err)
	assert.Nil(t, order)
	var apiErr *MannaiahAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode())
	assert.Contains(t, apiErr.Body, "Forbidden resource")
}

func writeJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}
