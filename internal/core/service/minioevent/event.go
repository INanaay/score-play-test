package minioevent

import (
	"log/slog"
	"score-play/internal/core/port"
)

type minioEventService struct {
	storage     port.FileStorage
	uof         port.UnitOfWork
	fileService port.FileService
	logger      *slog.Logger
}

// NewMinioEventService creates a new Minio event handler
func NewMinioEventService(storage port.FileStorage, uof port.UnitOfWork, fileService port.FileService, logger *slog.Logger) port.MessageService {
	return &minioEventService{
		storage:     storage,
		uof:         uof,
		fileService: fileService,
		logger:      logger,
	}
}
