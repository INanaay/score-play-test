package domain

import (
	"time"

	"github.com/google/uuid"
)

// FileStatus represents the status of a file
type FileStatus string

const (
	FileStatusUploading FileStatus = "uploading"
	FileStatusCompleted FileStatus = "completed"
	FileStatusFailed    FileStatus = "failed"
)

// FileType represents a file type
type FileType string

const (
	FileTypeImage   FileType = "image"
	FileTypeVideo   FileType = "video"
	FileTypeUnknown FileType = "unknown"
)

// FileMetadata represents a file metadata
type FileMetadata struct {
	ID         uuid.UUID
	Filename   string
	MimeType   string
	MediaType  string
	SizeBytes  int64
	StorageKey string
	Checksum   string
	Status     FileStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}
