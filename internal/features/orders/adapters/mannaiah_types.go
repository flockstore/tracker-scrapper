package adapter

import "time"

type mannaiahContactListResponse struct {
	Data []mannaiahContact `json:"data"`
}

type mannaiahContact struct {
	ID             string            `json:"id"`
	MongoID        string            `json:"_id"`
	Email          string            `json:"email"`
	FirstName      string            `json:"firstName"`
	LastName       string            `json:"lastName"`
	LegalName      string            `json:"legalName"`
	Address        string            `json:"address"`
	AddressExtra   string            `json:"addressExtra"`
	CityCode       string            `json:"cityCode"`
	Metadata       map[string]string `json:"metadata"`
	DocumentNumber string            `json:"documentNumber"`
}

type mannaiahOrderListResponse struct {
	Data []mannaiahOrder `json:"data"`
}

type mannaiahOrder struct {
	ID              string            `json:"id"`
	Identifier      string            `json:"identifier"`
	ContactID       string            `json:"contactId"`
	CurrentStatus   string            `json:"currentStatus"`
	PaymentMethod   string            `json:"paymentMethod"`
	CreatedAt       time.Time         `json:"createdAt"`
	Items           []mannaiahItem    `json:"items"`
	ShippingAddress mannaiahAddress   `json:"shippingAddress"`
	Metadata        map[string]string `json:"metadata"`
}

type mannaiahItem struct {
	Quantity      int    `json:"quantity"`
	SKU           string `json:"sku"`
	AlternateName string `json:"alternateName"`
	ProductID     string `json:"productId"`
}

type mannaiahAddress struct {
	Address  string `json:"address"`
	Address2 string `json:"address2"`
	CityCode string `json:"cityCode"`
	Phone    string `json:"phone"`
}

type mannaiahShippingMarkListResponse struct {
	Data []mannaiahShippingMark `json:"data"`
}

type mannaiahShippingMark struct {
	CarrierID      string `json:"carrierId"`
	TrackingNumber string `json:"trackingNumber"`
}

type mannaiahTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}
