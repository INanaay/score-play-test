package domain

import (
	"time"

	"github.com/google/uuid"
)

// Tag represents a tag entity
type Tag struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}
