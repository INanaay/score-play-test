package port

import (
	"context"
	"score-play/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

// UploadSessionRepository is an interface to interact with upload session repositories
type UploadSessionRepository interface {
	Create(ctx context.Context, session domain.UploadSession) error
	UpdateExpiresAt(ctx context.Context, id uuid.UUID, expiresAt time.Time) error
	FindByIDAndActive(ctx context.Context, id uuid.UUID) (*domain.UploadSession, error)
	UpdateStatusByFileID(ctx context.Context, fileID uuid.UUID, status domain.UploadSessionStatus) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.UploadSession, error)
	FindByFileID(ctx context.Context, fileID uuid.UUID) (*domain.UploadSession, error)
	FindAllExpired(ctx context.Context, now time.Time) ([]domain.UploadSession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UploadSessionStatus) error
}
