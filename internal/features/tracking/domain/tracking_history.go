package domain

import "time"

// TrackingStatus represents the current global status of a shipment.
type TrackingStatus string

const (
	// TrackingStatusProcessing indicates the shipment is being processed.
	TrackingStatusProcessing TrackingStatus = "PROCESSING"
	// TrackingStatusCompleted indicates the shipment has been delivered.
	TrackingStatusCompleted TrackingStatus = "COMPLETED"
	// TrackingStatusOrigin indicates the shipment is at the origin facility.
	TrackingStatusOrigin TrackingStatus = "ORIGIN"
	// TrackingStatusReturn indicates the shipment was returned to sender.
	TrackingStatusReturn TrackingStatus = "RETURN"
	// TrackingStatusIncidence indicates there is an issue with the shipment.
	TrackingStatusIncidence TrackingStatus = "INCIDENCE"
)

// TrackingHistory represents the complete tracking information for a shipment.
type TrackingHistory struct {
	// GlobalStatus is the overall status of the shipment.
	GlobalStatus TrackingStatus `json:"global_status"`
	// History contains the chronological events for the shipment.
	History []TrackingEvent `json:"history"`
}

// TrackingEvent represents a single event in the shipment's tracking history.
type TrackingEvent struct {
	// Date is the timestamp when the event occurred.
	Date time.Time `json:"date"`
	// Text is the description of the tracking event.
	Text string `json:"text"`
	// City is the location where the event occurred.
	City string `json:"city"`
	// Code is the courier-specific status code for this event.
	Code string `json:"code"`
}
