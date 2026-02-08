package service

import (
	"context"
	"errors"
	"testing"
	"tracker-scrapper/internal/features/banners/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBannerRepository is a mock implementation of ports.BannerRepository
type MockBannerRepository struct {
	mock.Mock
}

func (m *MockBannerRepository) Save(ctx context.Context, banner *domain.Banner) error {
	args := m.Called(ctx, banner)
	return args.Error(0)
}

func (m *MockBannerRepository) Get(ctx context.Context) (*domain.Banner, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Banner), args.Error(1)
}

func (m *MockBannerRepository) Delete(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestBannerService_SetBanner(t *testing.T) {
	mockRepo := new(MockBannerRepository)
	service := NewBannerService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.Banner")).Return(nil).Once()

		err := service.SetBanner(ctx, "Title", "Subtitle", domain.BannerTypeInfo, 60)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("InvalidType", func(t *testing.T) {
		err := service.SetBanner(ctx, "Title", "Subtitle", "INVALID", 60)
		assert.ErrorIs(t, err, domain.ErrInvalidBannerType)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.Banner")).Return(errors.New("db error")).Once()

		err := service.SetBanner(ctx, "Title", "Subtitle", domain.BannerTypeInfo, 60)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestBannerService_GetBanner(t *testing.T) {
	mockRepo := new(MockBannerRepository)
	service := NewBannerService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		expectedBanner := &domain.Banner{Title: "Test"}
		mockRepo.On("Get", ctx).Return(expectedBanner, nil).Once()

		banner, err := service.GetBanner(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedBanner, banner)
		mockRepo.AssertExpectations(t)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo.On("Get", ctx).Return(nil, errors.New("db error")).Once()

		banner, err := service.GetBanner(ctx)
		assert.Error(t, err)
		assert.Nil(t, banner)
		mockRepo.AssertExpectations(t)
	})
}

func TestBannerService_RemoveBanner(t *testing.T) {
	mockRepo := new(MockBannerRepository)
	service := NewBannerService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Delete", ctx).Return(nil).Once()

		err := service.RemoveBanner(ctx)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo.On("Delete", ctx).Return(errors.New("db error")).Once()

		err := service.RemoveBanner(ctx)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
