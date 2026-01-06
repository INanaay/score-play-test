package file

import (
	"log/slog"
	"score-play/internal/core/port"

	"github.com/go-chi/chi/v5"
)

// HandlerV1 is the handler for v1 tags routes
type HandlerV1 struct {
	fileService port.FileService
	logger      *slog.Logger
}

// NewFileHandlerV1 creates HandlerV1
func NewFileHandlerV1(service port.FileService, logger *slog.Logger) *HandlerV1 {
	return &HandlerV1{
		fileService: service,
		logger:      logger,
	}
}

// Routes exposes handler routes
func (h *HandlerV1) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/upload", h.UploadFileV1)
	router.Post("/upload/multipart", h.UploadFileMultipartV1)
	router.Post("/upload/multipart/{sessionID}/parts", h.RetrievePresignedPartsV1)
	router.Get("/upload/multipart/{sessionID}/parts", h.GetPartsV1)
	router.Post("/upload/multipart/{sessionID}/complete", h.CompleteMultipartV1)
	router.Get("/{fileID}/", h.GetFileV1)

	return router
}
