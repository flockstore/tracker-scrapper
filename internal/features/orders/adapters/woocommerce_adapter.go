package adapter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/core/httpclient"
	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/features/orders/domain"

	"go.uber.org/zap"
)

// WooCommerceAdapter implements the OrderProvider interface using the WooCommerce REST API.
type WooCommerceAdapter struct {
	// client is the HTTP client used for API requests.
	client *http.Client
	// config holds the WooCommerce connection details.
	config config.WooCommerceConfig
}

// NewWooCommerceAdapter creates a new instance of WooCommerceAdapter.
func NewWooCommerceAdapter(cfg config.WooCommerceConfig) *WooCommerceAdapter {
	return &WooCommerceAdapter{
		client: httpclient.NewClient(10 * time.Second),
		config: cfg,
	}
}

// GetOrder fetches an order from WooCommerce and maps it to the domain entity.
func (a *WooCommerceAdapter) GetOrder(orderID string) (*domain.Order, error) {
	url := fmt.Sprintf("%s/wp-json/wc/v3/orders/%s", a.config.URL, orderID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Basic Auth using optimized string building
	authVal := make([]byte, 0, len(a.config.ConsumerKey)+len(a.config.ConsumerSecret)+1)
	authVal = fmt.Appendf(authVal, "%s:%s", a.config.ConsumerKey, a.config.ConsumerSecret)

	encoded := base64.StdEncoding.EncodeToString(authVal)
	req.Header.Add("Authorization", "Basic "+encoded)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("order not found: %s", orderID)
		}
		return nil, fmt.Errorf("woocommerce API returned status: %d", resp.StatusCode)
	}

	var wcOrder woocommerceOrder
	if err := json.NewDecoder(resp.Body).Decode(&wcOrder); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return a.mapToDomain(wcOrder, orderID), nil
}

// HealthCheck verifies that the WooCommerce API is reachable and credentials are valid.
func (a *WooCommerceAdapter) HealthCheck() error {
	// Check orders endpoint with per_page=1 to verify auth and reachability
	url := fmt.Sprintf("%s/wp-json/wc/v3/orders?per_page=1", a.config.URL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("health check failed to create request: %w", err)
	}

	authVal := make([]byte, 0, len(a.config.ConsumerKey)+len(a.config.ConsumerSecret)+1)
	authVal = fmt.Appendf(authVal, "%s:%s", a.config.ConsumerKey, a.config.ConsumerSecret)
	encoded := base64.StdEncoding.EncodeToString(authVal)
	req.Header.Add("Authorization", "Basic "+encoded)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// mapToDomain converts a raw WooCommerce order response into a domain Order entity.
func (a *WooCommerceAdapter) mapToDomain(wcOrder woocommerceOrder, orderID string) *domain.Order {
	tracking := a.extractTrackingInfo(wcOrder, orderID)
	status := mapStatus(wcOrder.Status, tracking)

	return &domain.Order{
		ID:            strconv.Itoa(wcOrder.ID),
		Status:        status,
		FirstName:     wcOrder.Billing.FirstName,
		LastName:      wcOrder.Billing.LastName,
		Address:       wcOrder.Shipping.Address1,
		City:          wcOrder.Shipping.City,
		State:         wcOrder.Shipping.State,
		Email:         wcOrder.Billing.Email,
		PaymentMethod: wcOrder.PaymentMethodTitle,
		Tracking:      tracking,
		CreatedAt:     time.Time(wcOrder.DateCreated),
		Items:         mapItems(wcOrder.LineItems, wcOrder.FeeLines),
	}
}

// mapStatus determines the domain OrderStatus based on WooCommerce status and tracking info.
func mapStatus(status string, tracking []domain.TrackingInfo) domain.OrderStatus {
	if len(tracking) > 0 {
		return domain.OrderStatusShipped
	}

	lowerStatus := strings.ToLower(status)

	switch lowerStatus {
	case "completed":
		return domain.OrderStatusShipped
	case "cancelled", "refunded", "failed":
		return domain.OrderStatusCancelled
	case "pending", "processing", "on-hold":
		return domain.OrderStatusCreated
	default:
		return domain.OrderStatusPending
	}
}

// extractTrackingInfo attempts to find tracking information from order metadata.
func (a *WooCommerceAdapter) extractTrackingInfo(order woocommerceOrder, orderID string) []domain.TrackingInfo {
	var tracking []domain.TrackingInfo

	for _, shippingLine := range order.ShippingLines {
		var trackingNum, trackingProvider string

		for _, meta := range shippingLine.MetaData {
			switch meta.Key {
			case "Tracking Number", "tracking_number", "_tracking_number":
				if val, ok := meta.Value.(string); ok && val != "" {
					trackingNum = val
				}
			case "Tracking Company", "tracking_company", "_tracking_company", "tracking_provider":
				if val, ok := meta.Value.(string); ok && val != "" {
					trackingProvider = val
				}
			}
		}

		if trackingNum != "" || trackingProvider != "" {
			tracking = append(tracking, domain.TrackingInfo{
				TrackingNumber:   trackingNum,
				TrackingProvider: trackingProvider,
			})
		}
	}

	if len(tracking) > 0 {
		return tracking
	}

	for _, meta := range order.MetaData {
		if meta.Key == "_wc_shipment_tracking_items" {
			if items, err := parseTrackingItems(meta.Value); err == nil && len(items) > 0 {
				return items
			}
		}
	}

	var legacyNum, legacyProvider string
	for _, meta := range order.MetaData {
		if meta.Key == "tracking_number" || meta.Key == "_tracking_number" || meta.Key == "wc_shipment_tracking_number" {
			if val, ok := meta.Value.(string); ok && val != "" {
				legacyNum = val
			}
		}
		if meta.Key == "tracking_company" || meta.Key == "_tracking_company" || meta.Key == "tracking_provider" {
			if val, ok := meta.Value.(string); ok && val != "" {
				legacyProvider = val
			}
		}
	}

	if legacyNum != "" || legacyProvider != "" {
		tracking = append(tracking, domain.TrackingInfo{
			TrackingNumber:   legacyNum,
			TrackingProvider: legacyProvider,
		})
	}

	// Final fallback: fetch and parse order notes
	if len(tracking) == 0 {
		tracking = a.getTrackingFromNotes(orderID)
	}

	return tracking
}

// parseTrackingItems parses the WooCommerce tracking items JSON structure.
func parseTrackingItems(value interface{}) ([]domain.TrackingInfo, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var wcItems []wcTrackingItem
	if err := json.Unmarshal(jsonBytes, &wcItems); err != nil {
		return nil, err
	}

	var tracking []domain.TrackingInfo
	for _, item := range wcItems {
		tracking = append(tracking, domain.TrackingInfo{
			TrackingProvider: item.TrackingProvider,
			TrackingNumber:   item.TrackingNumber,
		})
	}

	return tracking, nil
}

// getTrackingFromNotes fetches order notes from WooCommerce API and extracts tracking information.
func (a *WooCommerceAdapter) getTrackingFromNotes(orderID string) []domain.TrackingInfo {
	url := fmt.Sprintf("%s/wp-json/wc/v3/orders/%s/notes", a.config.URL, orderID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Get().Warn("Failed to create notes request", zap.String("order_id", orderID), zap.Error(err))
		return nil
	}

	// Basic Auth
	authVal := make([]byte, 0, len(a.config.ConsumerKey)+len(a.config.ConsumerSecret)+1)
	authVal = fmt.Appendf(authVal, "%s:%s", a.config.ConsumerKey, a.config.ConsumerSecret)
	encoded := base64.StdEncoding.EncodeToString(authVal)
	req.Header.Add("Authorization", "Basic "+encoded)

	resp, err := a.client.Do(req)
	if err != nil {
		logger.Get().Warn("Failed to fetch order notes", zap.String("order_id", orderID), zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Get().Warn("Order notes request failed", zap.String("order_id", orderID), zap.Int("status", resp.StatusCode))
		return nil
	}

	var notes []wcOrderNote
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		logger.Get().Warn("Failed to decode order notes", zap.String("order_id", orderID), zap.Error(err))
		return nil
	}

	// Search for tracking info in customer notes
	for _, note := range notes {
		if note.CustomerNote && note.Note != "" {
			if tracking := extractTrackingFromNotes(note.Note); len(tracking) > 0 {
				return tracking
			}
		}
	}

	return nil
}

// extractTrackingFromNotes parses customer notes to extract tracking information.
// Matches patterns like: "No de guía: 2259176774 Paquetería: servientrega_co"
func extractTrackingFromNotes(notes string) []domain.TrackingInfo {
	if notes == "" {
		return nil
	}

	// Pattern matches: "No de guía: {number} Paquetería: {carrier}"
	// Case-insensitive, handles accents (guía/guia), flexible whitespace
	pattern := regexp.MustCompile(`(?i)no\s+de\s+gu[ií]a:\s*(\S+).*?paqueter[ií]a:\s*(\S+)`)
	matches := pattern.FindStringSubmatch(notes)

	if len(matches) < 3 {
		return nil
	}

	trackingNumber := strings.TrimSpace(matches[1])
	carrier := strings.TrimSpace(matches[2])

	// Normalize carrier name to standard format
	normalizedCarrier := normalizeCarrierName(carrier)

	if trackingNumber == "" || normalizedCarrier == "" {
		return nil
	}

	return []domain.TrackingInfo{
		{
			TrackingNumber:   trackingNumber,
			TrackingProvider: normalizedCarrier,
		},
	}
}

// normalizeCarrierName converts various carrier name formats to standardized format.
func normalizeCarrierName(carrier string) string {
	carrier = strings.ToLower(strings.TrimSpace(carrier))

	// Map common variations to standard format
	switch {
	case strings.Contains(carrier, "servientrega"):
		return "servientrega_co"
	case strings.Contains(carrier, "coordinadora"):
		return "coordinadora_co"
	case strings.Contains(carrier, "interrapidisimo") || strings.Contains(carrier, "inter"):
		return "interrapidisimo_co"
	default:
		// Return as-is if already in correct format or unknown
		if strings.HasSuffix(carrier, "_co") {
			return carrier
		}
		return carrier + "_co"
	}
}

// mapItems converts WooCommerce line items and fee lines to domain OrderItems.
func mapItems(wcItems []wcLineItem, feeLines []wcFeeLine) []domain.OrderItem {
	items := make([]domain.OrderItem, 0, len(wcItems)+len(feeLines))

	for _, item := range wcItems {
		var picture string
		if len(item.Image.Src) > 0 {
			picture = item.Image.Src
		}
		items = append(items, domain.OrderItem{
			Quantity: item.Quantity,
			SKU:      item.Sku,
			Name:     item.Name,
			Picture:  picture,
		})
	}

	for _, fee := range feeLines {
		items = append(items, domain.OrderItem{
			Quantity: 1,
			SKU:      "",
			Name:     fee.Name,
			Picture:  "",
		})
	}

	return items
}

// internal structs for mapping

// woocommerceOrder represents the JSON structure of an order from WooCommerce API.
type woocommerceOrder struct {
	// ID is the unique order ID.
	ID int `json:"id"`
	// Status is the order status (e.g., pending, processing, completed).
	Status string `json:"status"`
	// DateCreated is the timestamp when the order was created.
	DateCreated wcTime `json:"date_created"`
	// PaymentMethodTitle is the display name of the payment method.
	PaymentMethodTitle string `json:"payment_method_title"`
	// Billing holds the billing address details.
	Billing wcBilling `json:"billing"`
	// Shipping holds the shipping address details.
	Shipping wcShipping `json:"shipping"`
	// LineItems contains the products ordered.
	LineItems []wcLineItem `json:"line_items"`
	// FeeLines contains additional fees/products added to the order.
	FeeLines []wcFeeLine `json:"fee_lines"`
	// ShippingLines contains shipment information including tracking data.
	ShippingLines []wcShippingLine `json:"shipping_lines"`
	// MetaData contains extra fields.
	MetaData []wcMetaData `json:"meta_data"`
}

// wcMetaData represents a key-value pair in WooCommerce metadata.
type wcMetaData struct {
	// Key is the metadata key name.
	Key string `json:"key"`
	// Value is the metadata value, which can be of various types.
	Value interface{} `json:"value"`
}

// wcOrderNote represents a note from the WooCommerce order notes endpoint.
type wcOrderNote struct {
	// ID is the unique note ID.
	ID int `json:"id"`
	// Author is the note author.
	Author string `json:"author"`
	// Note is the note content.
	Note string `json:"note"`
	// CustomerNote indicates if this note is visible to customers.
	CustomerNote bool `json:"customer_note"`
	// DateCreated is when the note was created.
	DateCreated string `json:"date_created"`
}

// wcTrackingItem represents a single tracking entry from WooCommerce Shipment Tracking plugin.
type wcTrackingItem struct {
	// TrackingProvider is the carrier name.
	TrackingProvider string `json:"tracking_provider"`
	// TrackingNumber is the shipment tracking number.
	TrackingNumber string `json:"tracking_number"`
	// DateShipped is the date the package was shipped (format: YYYY-MM-DD).
	DateShipped string `json:"date_shipped"`
}

// wcBilling holds billing address information.
type wcBilling struct {
	// FirstName is the customer's first name.
	FirstName string `json:"first_name"`
	// LastName is the customer's last name.
	LastName string `json:"last_name"`
	// Email is the customer's email address.
	Email string `json:"email"`
}

// wcShipping holds shipping address information.
type wcShipping struct {
	// Address1 is the primary address line.
	Address1 string `json:"address_1"`
	// City is the shipping city.
	City string `json:"city"`
	// State is the shipping state or province.
	State string `json:"state"`
}

// wcLineItem represents a product in the WooCommerce order.
type wcLineItem struct {
	// ID is the unique identifier for the line item.
	ID int `json:"id"`
	// Name is the product name.
	Name string `json:"name"`
	// Sku is the product SKU.
	Sku string `json:"sku"`
	// Quantity is the number of units ordered.
	Quantity int `json:"quantity"`
	// Image holds the product image details.
	Image wcImage `json:"image"`
}

// wcFeeLine represents a fee or additional product line item.
type wcFeeLine struct {
	// Name is the fee/product name.
	Name string `json:"name"`
}

// wcShippingLine represents a shipping method with tracking metadata.
type wcShippingLine struct {
	// MethodID is the shipping method identifier.
	MethodID string `json:"method_id"`
	// MethodTitle is the shipping method display name.
	MethodTitle string `json:"method_title"`
	// MetaData contains tracking information.
	MetaData []wcMetaData `json:"meta_data"`
}

// wcImage holds the product image URL.
type wcImage struct {
	// Src is the source URL of the image.
	Src string `json:"src"`
}

// wcTime is a custom helper struct to handle WooCommerce's date format.
type wcTime time.Time

// UnmarshalJSON parses the custom date format used by WooCommerce.
func (t *wcTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	// WooCommerce usually returns ISO8601 "2018-12-19T14:48:25"
	if s == "null" {
		*t = wcTime(time.Time{})
		return nil
	}
	parsed, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		// Try with timezone just in case
		parsed, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		// Log warning but don't fail? Or fail.
		logger.Get().Warn("Failed to parse date", zap.String("date", s), zap.Error(err))
		return nil // Return zero time
	}
	*t = wcTime(parsed)
	return nil
}
