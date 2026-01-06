package port

import (
	"context"
	"score-play/internal/core/domain"

	"github.com/google/uuid"
)

// TagRepository represents a tag repository implementation
type TagRepository interface {
	CreateMany(ctx context.Context, tags []string) (int, error)
	FindByName(ctx context.Context, name string) (*domain.Tag, error)
	FindByNames(ctx context.Context, names []string) (map[string]uuid.UUID, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Tag, error)
	List(ctx context.Context, limit int, marker *string) ([]domain.Tag, *string, error)
}

// TagService represents a tag service implementation
type TagService interface {
	CreateTags(ctx context.Context, name []string) error
	GetTagByName(ctx context.Context, name string) (*domain.Tag, error)
	ListTags(ctx context.Context, limit int, marker *string) ([]domain.Tag, *string, error)
}
