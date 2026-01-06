package cleanup

import (
	"log/slog"
	"score-play/internal/core/port"
)

type cleanupService struct {
	uow         port.UnitOfWork
	fileStorage port.FileStorage
	logger      *slog.Logger
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(uow port.UnitOfWork, fileStorage port.FileStorage, logger *slog.Logger) port.CleanupService {
	return &cleanupService{
		uow:         uow,
		fileStorage: fileStorage,
		logger:      logger,
	}
}
