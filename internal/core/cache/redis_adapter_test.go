package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisAdapter_GetSet(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	adapter, err := NewRedisAdapter("redis://" + mr.Addr())
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()

	// Test Set and Get
	key := "test_key"
	value := []byte("test_value")
	ttl := 10 * time.Second

	err = adapter.Set(ctx, key, value, ttl)
	assert.NoError(t, err)

	retrievedValue, err := adapter.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrievedValue)
}

func TestRedisAdapter_GetNotFound(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	adapter, err := NewRedisAdapter("redis://" + mr.Addr())
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()

	_, err = adapter.Get(ctx, "non_existent_key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

func TestRedisAdapter_Delete(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	adapter, err := NewRedisAdapter("redis://" + mr.Addr())
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()

	// Set a key
	key := "delete_test"
	err = adapter.Set(ctx, key, []byte("value"), 0)
	require.NoError(t, err)

	// Delete it
	err = adapter.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify it's gone
	_, err = adapter.Get(ctx, key)
	assert.Error(t, err)
}

func TestRedisAdapter_TTL(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	adapter, err := NewRedisAdapter("redis://" + mr.Addr())
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()

	// Set with short TTL
	key := "ttl_test"
	value := []byte("expires_soon")

	err = adapter.Set(ctx, key, value, 1*time.Second)
	require.NoError(t, err)

	// Should exist immediately
	_, err = adapter.Get(ctx, key)
	assert.NoError(t, err)

	// Fast forward time in miniredis
	mr.FastForward(2 * time.Second)

	// Should be expired
	_, err = adapter.Get(ctx, key)
	assert.Error(t, err)
}

func TestRedisAdapter_Ping(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	adapter, err := NewRedisAdapter("redis://" + mr.Addr())
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()

	err = adapter.Ping(ctx)
	assert.NoError(t, err)
}

func TestRedisAdapter_InvalidURL(t *testing.T) {
	_, err := NewRedisAdapter("invalid://url")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse Redis URL")
}
