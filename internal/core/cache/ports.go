package cache

import (
	"context"
	"time"
)

// Cache defines the caching operations interface following hexagonal architecture.
// This is a port that can be implemented by different cache providers (Redis, Memcached, etc.).
type Cache interface {
	// Get retrieves a value from the cache by key.
	// Returns the cached value or an error if not found or on failure.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with the specified key and TTL.
	// TTL of 0 means no expiration.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from the cache by key.
	Delete(ctx context.Context, key string) error

	// Ping checks if the cache service is reachable.
	Ping(ctx context.Context) error

	// Close closes the cache connection.
	Close() error
}
