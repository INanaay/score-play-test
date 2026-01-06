package port

import (
	"context"
	"score-play/internal/core/domain"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// FileRepository is an interface to define file repository interactions
type FileRepository interface {
	Create(ctx context.Context, id uuid.UUID, fileName, mimeType string, mediaType domain.FileType, size int64, status domain.FileStatus, checksum string, storageKey string) error
	FindById(ctx context.Context, id uuid.UUID) (*domain.FileMetadata, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindExpired(ctx context.Context, expirationTime time.Time) ([]domain.FileMetadata, error)
}

// FileStorage is an interface to define file storage interactions
type FileStorage interface {
	GeneratePresignedURLSimpleUpload(ctx context.Context, fileKey string, checksumSha256 string) (string, map[string]string, *time.Time, error)
	InitMultipartUpload(ctx context.Context, fileName string, checksum string) (string, error)
	GeneratePresignedURLForPart(ctx context.Context, fileKey string, partNumber int, uploadID, mimeType string, contentLength int64, checksumSha256 string) (string, map[string]string, *time.Time, error)
	CompleteMultipartUpload(ctx context.Context, fileName string, uploadID string, parts []domain.UploadPart) error
	GetObjectInfo(ctx context.Context, fileKey string) (*minio.ObjectInfo, error)
	ListPartsPaginated(ctx context.Context, fileKey string, uploadID string, maxParts int, partNumberMarker int) ([]domain.UploadPart, int, error)
	AbortMultipartUpload(ctx context.Context, fileKey string, uploadID string) error
	DeleteObject(ctx context.Context, fileKey string) error
	GeneratePresignedURLForDownload(ctx context.Context, fileKey string) (string, *time.Time, error)
	GetHeaderBytes(ctx context.Context, fileKey string, n int64) ([]byte, error)
}

// FileService is an interface to define file service
type FileService interface {
	RequestUploadFile(ctx context.Context, fileName string, contentType string, sizeBytes int64, checksumSha256 string, tags []string) (*uuid.UUID, *string, map[string]string, *time.Time, error)
	RequestUploadMultipartFile(ctx context.Context, fileName string, contentType string, sizeBytes int64, checksumSha256 string, tags []string) (*uuid.UUID, int, error)
	GetPresignedParts(ctx context.Context, sessionID uuid.UUID, parts []domain.UploadPart) ([]domain.UploadPart, error)
	ListParts(ctx context.Context, sessionID uuid.UUID, maxParts int, partNumberMarker int) ([]domain.UploadPart, int, error)
	CompleteMultipartUpload(ctx context.Context, sessionID uuid.UUID, parts []domain.UploadPart) (*uuid.UUID, error)
	GetFile(ctx context.Context, fileID uuid.UUID) (url *string, filename *string, tags []domain.Tag, expiresAt *time.Time, error error)
	FinalizeUpload(ctx context.Context, metadata domain.FileMetadata, err error, eventType domain.EventType) error
}
