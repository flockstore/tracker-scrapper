package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisAdapter implements the Cache interface using Redis.
type RedisAdapter struct {
	client *redis.Client
}

// NewRedisAdapter creates a new Redis cache adapter.
// The redisURL should be in the format: redis://[:password@]host[:port][/database]
func NewRedisAdapter(redisURL string) (*RedisAdapter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	return &RedisAdapter{client: client}, nil
}

// Get retrieves a value from Redis by key.
func (r *RedisAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in Redis with the specified TTL.
func (r *RedisAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Delete removes a value from Redis by key.
func (r *RedisAdapter) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Ping checks if Redis is reachable.
func (r *RedisAdapter) Ping(ctx context.Context) error {
	err := r.client.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// Close closes the Redis connection.
func (r *RedisAdapter) Close() error {
	return r.client.Close()
}
