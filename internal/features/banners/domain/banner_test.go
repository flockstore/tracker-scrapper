package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBanner(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		subtitle    string
		bannerType  BannerType
		duration    int
		expectedErr error
	}{
		{
			name:       "Valid INFO Banner",
			title:      "Info Title",
			subtitle:   "Info Subtitle",
			bannerType: BannerTypeInfo,
			duration:   60,
		},
		{
			name:       "Valid WARNING Banner",
			title:      "Warning Title",
			subtitle:   "Warning Subtitle",
			bannerType: BannerTypeWarning,
			duration:   0,
		},
		{
			name:       "Valid DANGER Banner",
			title:      "Danger Title",
			subtitle:   "Danger Subtitle",
			bannerType: BannerTypeDanger,
			duration:   120,
		},
		{
			name:        "Invalid Banner Type",
			title:       "Invalid",
			subtitle:    "Invalid",
			bannerType:  "INVALID",
			duration:    60,
			expectedErr: ErrInvalidBannerType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			banner, err := NewBanner(tt.title, tt.subtitle, tt.bannerType, tt.duration)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				assert.Nil(t, banner)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, banner)
				assert.Equal(t, tt.title, banner.Title)
				assert.Equal(t, tt.subtitle, banner.Subtitle)
				assert.Equal(t, tt.bannerType, banner.Type)
				assert.Equal(t, tt.duration, banner.Duration)
				assert.False(t, banner.CreatedAt.IsZero())
			}
		})
	}
}
