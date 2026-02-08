package domain

import (
	"errors"
	"time"
)

// BannerType represents the severity/type of the banner.
type BannerType string

const (
	BannerTypeInfo    BannerType = "INFO"
	BannerTypeWarning BannerType = "WARNING"
	BannerTypeDanger  BannerType = "DANGER"
)

var (
	ErrInvalidBannerType = errors.New("invalid banner type")
)

// Banner represents a site-wide alert.
type Banner struct {
	Title     string     `json:"title"`
	Subtitle  string     `json:"subtitle"`
	Type      BannerType `json:"type"`
	Duration  int        `json:"duration,omitempty"` // Duration in seconds. 0 means permanent (until manually deleted).
	CreatedAt time.Time  `json:"created_at"`
}

// NewBanner creates a new Banner and validates it.
func NewBanner(title, subtitle string, bannerType BannerType, duration int) (*Banner, error) {
	if bannerType != BannerTypeInfo && bannerType != BannerTypeWarning && bannerType != BannerTypeDanger {
		return nil, ErrInvalidBannerType
	}

	return &Banner{
		Title:     title,
		Subtitle:  subtitle,
		Type:      bannerType,
		Duration:  duration,
		CreatedAt: time.Now(),
	}, nil
}
