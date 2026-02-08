package ports

import (
	"context"
	"tracker-scrapper/internal/features/banners/domain"
)

// BannerService defines the primary port for banner operations.
type BannerService interface {
	SetBanner(ctx context.Context, title, subtitle string, bannerType domain.BannerType, duration int) error
	GetBanner(ctx context.Context) (*domain.Banner, error)
	RemoveBanner(ctx context.Context) error
}

// BannerRepository defines the secondary port for banner storage.
type BannerRepository interface {
	Save(ctx context.Context, banner *domain.Banner) error
	Get(ctx context.Context) (*domain.Banner, error)
	Delete(ctx context.Context) error
}
