package domain

import (
	"time"

	"github.com/google/uuid"
)

// UploadSessionStatus represents the status of an upload session
type UploadSessionStatus string

const (
	UploadSessionStatusOpen      UploadSessionStatus = "open"
	UploadSessionStatusCompleted UploadSessionStatus = "completed"
	UploadSessionStatusAborted   UploadSessionStatus = "aborted"
)

// UploadSession represents an upload session
type UploadSession struct {
	ID               uuid.UUID
	FileID           uuid.UUID
	ProviderUploadID string
	PartSize         int
	ExpiresAt        time.Time
	Status           UploadSessionStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// UploadPart represents an upload part (chunk)
type UploadPart struct {
	PartNumber     int
	ETag           string
	ChecksumSHA256 string
	PresignedURL   string
	ContentLength  int64
	Headers        map[string]string
	ExpiresAt      *time.Time
}
