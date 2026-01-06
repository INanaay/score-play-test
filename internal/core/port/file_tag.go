package port

import (
	"context"
	"score-play/internal/core/domain"

	"github.com/google/uuid"
)

type FileTagRepository interface {
	Create(ctx context.Context, fileID uuid.UUID, tagID uuid.UUID) error
	FindByFileID(ctx context.Context, fileID uuid.UUID) ([]domain.FileTag, error)
	CreateMany(ctx context.Context, fileID uuid.UUID, tagIDs []uuid.UUID) (int, error)
	DeleteByFileID(ctx context.Context, fileID uuid.UUID) error
}
