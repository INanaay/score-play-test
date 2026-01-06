package file

import (
	"encoding/json"
	"errors"
	"net/http"
	"score-play/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

// V1UploadFileRequest is the request to upload a small file
type V1UploadFileRequest struct {
	FileName       string   `json:"filename"`
	ContentType    string   `json:"content_type"`
	SizeBytes      int64    `json:"size_bytes"`
	ChecksumSha256 string   `json:"checksum_sha256"`
	Tags           []string `json:"tags"`
}

// V1UploadFileResponse is the response to upload a small file
type V1UploadFileResponse struct {
	FileID       uuid.UUID         `json:"file_id"`
	PresignedURL string            `json:"presigned_url"`
	Headers      map[string]string `json:"headers"`
	ExpiresAt    *time.Time        `json:"expires_at"`
}

func (h *HandlerV1) UploadFileV1(w http.ResponseWriter, r *http.Request) {

	var req V1UploadFileRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("error decoding upload file request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.FileName == "" || req.ContentType == "" || req.SizeBytes == 0 || req.ChecksumSha256 == "" {
		http.Error(w, "missing param", http.StatusBadRequest)
		return
	}

	if req.Tags == nil || len(req.Tags) == 0 {
		http.Error(w, "provide at least one tag", http.StatusBadRequest)
		return
	}

	fileID, presignedURL, headers, expiresAt, requestErr := h.fileService.RequestUploadFile(r.Context(), req.FileName, req.ContentType, req.SizeBytes, req.ChecksumSha256, req.Tags)
	switch {
	case errors.Is(requestErr, domain.ErrInvalidFileType), errors.Is(requestErr, domain.ErrFileSizeTooBig), errors.Is(requestErr, domain.ErrTagNotFound):
		h.logger.Error("invalid request", "error", requestErr)
		http.Error(w, requestErr.Error(), http.StatusBadRequest)
		return
	case requestErr != nil:
		h.logger.Error("error requesting presigned url", "error", err)
		http.Error(w, "internal server error", http.StatusServiceUnavailable)
		return
	default:
		var finalFileID uuid.UUID
		if fileID != nil {
			finalFileID = *fileID
		}

		var finalPresignedURL string
		if presignedURL != nil {
			finalPresignedURL = *presignedURL
		}

		resp := V1UploadFileResponse{
			FileID:       finalFileID,
			PresignedURL: finalPresignedURL,
			Headers:      headers,
			ExpiresAt:    expiresAt,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			h.logger.Error("error encoding response", "error", err)
		}
		return
	}

}
