package file

import (
	"encoding/json"
	"errors"
	"net/http"
	"score-play/internal/core/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CompletedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
	Checksum   string `json:"checksum"`
}

type V1CompleteMultipartRequest struct {
	Parts []CompletedPart `json:"parts"`
}

type V1CompleteMultipartResponse struct {
	FileID uuid.UUID `json:"file_id"`
}

func (h *HandlerV1) CompleteMultipartV1(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}
	uuidSession, parseErr := uuid.Parse(sessionID)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}

	var req V1CompleteMultipartRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("error decoding complete multipart request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Parts == nil || len(req.Parts) == 0 {
		http.Error(w, "Request contains no parts", http.StatusBadRequest)
		return
	}

	var domainParts []domain.UploadPart
	for _, part := range req.Parts {
		domainParts = append(domainParts, domain.UploadPart{
			PartNumber:     part.PartNumber,
			ETag:           part.ETag,
			ChecksumSHA256: part.Checksum,
		})
	}

	fileID, err := h.fileService.CompleteMultipartUpload(r.Context(), uuidSession, domainParts)
	switch {
	case errors.Is(err, domain.ErrSessionNotFound):
		http.Error(w, "Session not found", http.StatusForbidden)
		return
	case errors.Is(err, domain.ErrMismatchETag), errors.Is(err, domain.ErrMismatchNBParts), errors.Is(err, domain.ErrDuplicatePart):
		http.Error(w, "invalid part", http.StatusBadRequest)
		return
	case errors.Is(err, domain.ErrFileMetadataNotFound):
		http.Error(w, "file metadata not found", http.StatusNotFound)
		return
	case err != nil:
		h.logger.Error("error completing multipart upload", "error", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	case fileID == nil:
		h.logger.Error("file id is nil", "error", err)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	default:
		resp := V1CompleteMultipartResponse{
			FileID: *fileID,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			h.logger.Error("error encoding response", "error", err)
		}
		return
	}
}
