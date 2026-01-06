package tag

import (
	"log/slog"
	"score-play/internal/core/port"

	"github.com/go-chi/chi/v5"
)

// HandlerV1 is the handler for v1 tags routes
type HandlerV1 struct {
	tagService port.TagService
	logger     *slog.Logger
}

// NewTagHandlerV1 creates HandlerV1
func NewTagHandlerV1(service port.TagService, logger *slog.Logger) *HandlerV1 {
	return &HandlerV1{
		tagService: service,
		logger:     logger,
	}
}

// Routes exposes routes
func (h *HandlerV1) Routes() chi.Router {
	router := chi.NewRouter()

	router.Post("/", h.CreateTagsV1)
	router.Get("/", h.ListTagsV1)

	return router
}
