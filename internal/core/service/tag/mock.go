package tag

import (
	"context"
	"score-play/internal/core/domain"

	"github.com/stretchr/testify/mock"
)

// MockTagService is a mock implementation of TagService
type MockTagService struct {
	mock.Mock
}

func (m *MockTagService) ListTags(ctx context.Context, limit int, marker *string) ([]domain.Tag, *string, error) {
	args := m.Called(ctx, limit, marker)
	return args.Get(0).([]domain.Tag), args.Get(1).(*string), args.Error(2)
}

func (m *MockTagService) CreateTags(ctx context.Context, tags []string) error {
	args := m.Called(ctx, tags)
	return args.Error(0)
}

func (m *MockTagService) GetTagByName(ctx context.Context, name string) (*domain.Tag, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*domain.Tag), args.Error(1)
}

func (m *MockTagService) CreateTag(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}
