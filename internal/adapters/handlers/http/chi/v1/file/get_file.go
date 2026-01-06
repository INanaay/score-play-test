package file

import (
	"encoding/json"
	"errors"
	"net/http"
	"score-play/internal/core/domain"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// V1GetFileResponse is the response to get file
type V1GetFileResponse struct {
	Filename  string    `json:"filename"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	Tags      []string  `json:"tags"`
}

// GetFileV1 is the function that handles GetFile
func (h *HandlerV1) GetFileV1(w http.ResponseWriter, r *http.Request) {

	fileID := chi.URLParam(r, "fileID")
	if fileID == "" {
		http.Error(w, "file id is required", http.StatusBadRequest)
		return
	}
	uuidFileID, parseErr := uuid.Parse(fileID)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}

	url, filename, tags, expiresAt, err := h.fileService.GetFile(r.Context(), uuidFileID)
	switch {
	case errors.Is(err, domain.ErrFileNotReady):
		http.Error(w, "file not ready", http.StatusConflict)
		return
	case errors.Is(err, domain.ErrFileUploadFailed):
		http.Error(w, "file upload failed", http.StatusConflict)
		return
	case err != nil:
		h.logger.Error("error getting file", "error", err)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	case url == nil || filename == nil || tags == nil || expiresAt == nil:
		h.logger.Error("response has nil values", "url", url, "filename", filename, "tags", tags)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	default:
		var respTags []string
		for _, tag := range tags {
			respTags = append(respTags, tag.Name)
		}
		resp := V1GetFileResponse{
			Filename:  *filename,
			URL:       *url,
			ExpiresAt: *expiresAt,
			Tags:      respTags,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			h.logger.Error("error encoding response", "error", err)
		}
		return
	}
}
