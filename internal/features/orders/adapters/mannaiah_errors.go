package adapter

import "fmt"

// MannaiahAPIError describes a non-2xx response from the Mannaiah backend.
type MannaiahAPIError struct {
	// Status is the upstream HTTP status code.
	Status int
	// Endpoint is the sanitized upstream endpoint path.
	Endpoint string
	// Body is a bounded upstream response body excerpt.
	Body string
}

// Error returns a safe error message for callers.
func (e *MannaiahAPIError) Error() string {
	return fmt.Sprintf("mannaiah API returned status %d for %s", e.Status, e.Endpoint)
}

// StatusCode returns the upstream HTTP status code.
func (e *MannaiahAPIError) StatusCode() int {
	return e.Status
}
