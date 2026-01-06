package port

import (
	"context"
	"time"
)

// CleanupService is service that handles cleanup
type CleanupService interface {
	CleanupExpiredFiles(ctx context.Context, now time.Time) error
	CleanupExpiredSessions(ctx context.Context, now time.Time) error
}
