package service

import (
	"context"
	"fmt"

	"tracker-scrapper/internal/features/banners/domain"
	"tracker-scrapper/internal/features/banners/ports"
)

// BannerServiceImpl implements ports.BannerService.
type BannerServiceImpl struct {
	repo ports.BannerRepository
}

// NewBannerService creates a new BannerServiceImpl.
func NewBannerService(repo ports.BannerRepository) *BannerServiceImpl {
	return &BannerServiceImpl{
		repo: repo,
	}
}

// SetBanner creates and saves a new banner.
func (s *BannerServiceImpl) SetBanner(ctx context.Context, title, subtitle string, bannerType domain.BannerType, duration int) error {
	banner, err := domain.NewBanner(title, subtitle, bannerType, duration)
	if err != nil {
		return err
	}

	if err := s.repo.Save(ctx, banner); err != nil {
		return fmt.Errorf("service: failed to save banner: %w", err)
	}

	return nil
}

// GetBanner retrieves the current banner.
func (s *BannerServiceImpl) GetBanner(ctx context.Context) (*domain.Banner, error) {
	banner, err := s.repo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get banner: %w", err)
	}

	return banner, nil
}

// RemoveBanner deletes the current banner.
func (s *BannerServiceImpl) RemoveBanner(ctx context.Context) error {
	if err := s.repo.Delete(ctx); err != nil {
		return fmt.Errorf("service: failed to remove banner: %w", err)
	}

	return nil
}
