package domain

import "github.com/google/uuid"

// FileTag represents a FileTag entity
type FileTag struct {
	FileID uuid.UUID
	TagID  uuid.UUID
}
