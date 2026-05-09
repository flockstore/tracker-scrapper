package adapter

import (
	"strings"

	"tracker-scrapper/internal/features/orders/domain"
)

func mapMannaiahOrder(order mannaiahOrder, contact mannaiahContact, tracking []domain.TrackingInfo) *domain.Order {
	shippingAddress := order.ShippingAddress.Address
	if shippingAddress == "" {
		shippingAddress = contact.Address
	}

	cityCode := order.ShippingAddress.CityCode
	if cityCode == "" {
		cityCode = contact.CityCode
	}

	firstName := contact.FirstName
	lastName := contact.LastName
	if firstName == "" && lastName == "" {
		firstName, lastName = splitName(contact.LegalName)
	}

	return &domain.Order{
		ID:            order.Identifier,
		Status:        mapMannaiahStatus(order.CurrentStatus, tracking),
		FirstName:     firstName,
		LastName:      lastName,
		Address:       shippingAddress,
		City:          cityCode,
		State:         "",
		Email:         contact.Email,
		PaymentMethod: order.PaymentMethod,
		Tracking:      tracking,
		CreatedAt:     order.CreatedAt,
		Items:         mapMannaiahItems(order.Items),
	}
}

func mapMannaiahStatus(status string, tracking []domain.TrackingInfo) domain.OrderStatus {
	if len(tracking) > 0 {
		return domain.OrderStatusShipped
	}

	switch strings.ToUpper(status) {
	case "COMPLETED":
		return domain.OrderStatusShipped
	case "CANCELLED":
		return domain.OrderStatusCancelled
	case "PENDING", "HOLD":
		return domain.OrderStatusPending
	default:
		return domain.OrderStatusCreated
	}
}

func mapMannaiahItems(items []mannaiahItem) []domain.OrderItem {
	mapped := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, domain.OrderItem{
			Quantity: item.Quantity,
			SKU:      item.SKU,
			Name:     item.displayName(),
			Picture:  "",
		})
	}
	return mapped
}

func (c mannaiahContact) entityID() string {
	if c.ID != "" {
		return c.ID
	}
	return c.MongoID
}

func (i mannaiahItem) displayName() string {
	if i.AlternateName != "" {
		return i.AlternateName
	}
	if i.SKU != "" {
		return i.SKU
	}
	return i.ProductID
}

func splitName(name string) (string, string) {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func orderIdentifierCandidates(identifier string) []string {
	trimmed := strings.TrimSpace(identifier)
	withoutHash := strings.TrimLeft(trimmed, "#")
	candidates := []string{trimmed, withoutHash}

	if withoutHash != "" {
		candidates = append(candidates, "#"+withoutHash)
	}

	seen := make(map[string]struct{}, len(candidates))
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}

	return unique
}

func sameOrderIdentifier(storedIdentifier string, requestedIdentifier string) bool {
	return normalizeOrderIdentifier(storedIdentifier) == normalizeOrderIdentifier(requestedIdentifier)
}

func normalizeOrderIdentifier(identifier string) string {
	return strings.ToLower(strings.TrimLeft(strings.TrimSpace(identifier), "#"))
}
