package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/features/orders/domain"
	"tracker-scrapper/internal/features/orders/ports"

	"go.uber.org/zap"
)

// MannaiahAdapter implements the OrderProvider interface using the Mannaiah API.
type MannaiahAdapter struct {
	// client is the HTTP client used for API requests.
	client *http.Client
	// config holds the Mannaiah connection details.
	config config.MannaiahConfig
	// token caches M2M access tokens until shortly before expiry.
	token mannaiahTokenCache
}

// NewMannaiahAdapter creates a new instance of MannaiahAdapter.
func NewMannaiahAdapter(cfg config.MannaiahConfig) *MannaiahAdapter {
	return &MannaiahAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
		config: cfg,
	}
}

// GetOrder fetches an order from Mannaiah and maps it to the tracker domain entity.
func (a *MannaiahAdapter) GetOrder(orderID string, email string) (*domain.Order, error) {
	contact, err := a.findContactByEmail(email)
	if err != nil {
		return nil, err
	}

	if contact == nil {
		return nil, ports.ErrOrderNotFound
	}

	order, err := a.findOrder(orderID, contact.entityID())
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, ports.ErrOrderNotFound
	}

	tracking, err := a.findShippingMarks(order.ID)
	if err != nil {
		return nil, err
	}

	return mapMannaiahOrder(*order, *contact, tracking), nil
}

// HealthCheck verifies that the Mannaiah API is reachable and credentials are valid.
func (a *MannaiahAdapter) HealthCheck() error {
	var response mannaiahOrderListResponse
	if err := a.getJSON("/orders", map[string]string{"limit": "1"}, &response); err != nil {
		return fmt.Errorf("mannaiah health check failed: %w", err)
	}
	return nil
}

func (a *MannaiahAdapter) findContactByEmail(email string) (*mannaiahContact, error) {
	var response mannaiahContactListResponse
	if err := a.getJSON("/contacts", map[string]string{
		"email": email,
		"limit": "1",
	}, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, nil
	}

	return &response.Data[0], nil
}

func (a *MannaiahAdapter) findOrder(identifier string, contactID string) (*mannaiahOrder, error) {
	var response mannaiahOrderListResponse
	if err := a.getJSON("/orders", map[string]string{
		"identifier": identifier,
		"contactId":  contactID,
		"limit":      "1",
	}, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, nil
	}

	return &response.Data[0], nil
}

func (a *MannaiahAdapter) findShippingMarks(orderID string) ([]domain.TrackingInfo, error) {
	var response mannaiahShippingMarkListResponse
	if err := a.getJSON("/shipping/marks", map[string]string{
		"orderID": orderID,
		"limit":   "20",
	}, &response); err != nil {
		return nil, err
	}

	tracking := make([]domain.TrackingInfo, 0, len(response.Data))
	for _, mark := range response.Data {
		if mark.TrackingNumber == "" && mark.CarrierID == "" {
			continue
		}
		tracking = append(tracking, domain.TrackingInfo{
			TrackingProvider: mark.CarrierID,
			TrackingNumber:   mark.TrackingNumber,
		})
	}

	return tracking, nil
}

func (a *MannaiahAdapter) getJSON(path string, query map[string]string, target interface{}) error {
	reqURL, err := a.requestURL(path, query)
	if err != nil {
		return err
	}

	token, err := a.bearerToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create Mannaiah request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	logger.Get().Debug("Mannaiah request started",
		zap.String("method", http.MethodGet),
		zap.String("endpoint", path),
	)

	resp, err := a.client.Do(req)
	if err != nil {
		logger.Get().Error("Mannaiah request failed",
			zap.String("method", http.MethodGet),
			zap.String("endpoint", path),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to execute Mannaiah request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body := readBodyExcerpt(resp.Body)
		logLevel := logger.Get().Warn
		if resp.StatusCode >= http.StatusInternalServerError {
			logLevel = logger.Get().Error
		}
		logLevel("Mannaiah request returned non-success status",
			zap.String("method", http.MethodGet),
			zap.String("endpoint", path),
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", body),
			zap.Duration("duration", time.Since(start)),
		)
		return &MannaiahAPIError{
			Status:   resp.StatusCode,
			Endpoint: path,
			Body:     body,
		}
	}

	logger.Get().Debug("Mannaiah request completed",
		zap.String("method", http.MethodGet),
		zap.String("endpoint", path),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", time.Since(start)),
	)

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode Mannaiah response: %w", err)
	}

	return nil
}

func (a *MannaiahAdapter) requestURL(path string, query map[string]string) (string, error) {
	base, err := url.Parse(strings.TrimRight(a.config.BackendURL, "/") + path)
	if err != nil {
		return "", fmt.Errorf("invalid Mannaiah URL: %w", err)
	}

	values := base.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	base.RawQuery = values.Encode()

	return base.String(), nil
}

func readBodyExcerpt(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 2048))
	if err != nil {
		return ""
	}
	return string(data)
}

func sanitizedURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "invalid-url"
	}
	parsed.RawQuery = ""
	parsed.User = nil
	return parsed.String()
}
