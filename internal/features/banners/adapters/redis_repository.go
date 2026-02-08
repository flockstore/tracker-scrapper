package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tracker-scrapper/internal/core/cache"
	"tracker-scrapper/internal/features/banners/domain"
)

const bannerCacheKey = "site_banner"

// RedisBannerRepository implements ports.BannerRepository using the cache adaptation.
type RedisBannerRepository struct {
	cache cache.Cache
}

// NewRedisBannerRepository creates a new RedisBannerRepository.
func NewRedisBannerRepository(c cache.Cache) *RedisBannerRepository {
	return &RedisBannerRepository{
		cache: c,
	}
}

// Save stores the banner in the cache.
func (r *RedisBannerRepository) Save(ctx context.Context, banner *domain.Banner) error {
	data, err := json.Marshal(banner)
	if err != nil {
		return fmt.Errorf("failed to marshal banner: %w", err)
	}

	ttl := time.Duration(banner.Duration) * time.Second
	// If duration is 0, it means permanent, so we pass 0 which cache treats as no expiration.

	if err := r.cache.Set(ctx, bannerCacheKey, data, ttl); err != nil {
		return fmt.Errorf("failed to save banner to cache: %w", err)
	}

	return nil
}

// Get retrieves the banner from the cache.
func (r *RedisBannerRepository) Get(ctx context.Context) (*domain.Banner, error) {
	data, err := r.cache.Get(ctx, bannerCacheKey)
	if err != nil {
		// Check if the error is due to key not found
		if err.Error() == fmt.Sprintf("key not found: %s", bannerCacheKey) {
			return nil, nil // Return nil, nil to indicate not found
		}
		return nil, fmt.Errorf("failed to get banner from cache: %w", err)
	}
	if data == nil {
		return nil, nil // No banner found
	}

	var banner domain.Banner
	if err := json.Unmarshal(data, &banner); err != nil {
		return nil, fmt.Errorf("failed to unmarshal banner: %w", err)
	}

	return &banner, nil
}

// Delete removes the banner from the cache.
func (r *RedisBannerRepository) Delete(ctx context.Context) error {
	if err := r.cache.Delete(ctx, bannerCacheKey); err != nil {
		return fmt.Errorf("failed to delete banner from cache: %w", err)
	}
	return nil
}
