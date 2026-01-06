package file

import (
	"encoding/json"
	"errors"
	"net/http"
	"score-play/internal/core/domain"

	"github.com/google/uuid"
)

// V1UploadMultipartRequest is the request to upload a multipart file
type V1UploadMultipartRequest struct {
	FileName       string   `json:"filename"`
	ContentType    string   `json:"content_type"`
	SizeBytes      int64    `json:"size_bytes"`
	ChecksumSha256 string   `json:"checksum_sha256"`
	Tags           []string `json:"tags"`
}

// V1UploadMultipartResponse is the response to upload a multipart file
type V1UploadMultipartResponse struct {
	SessionID uuid.UUID `json:"session_id"`
	PartSize  int       `json:"part_size"`
}

func (h *HandlerV1) UploadFileMultipartV1(w http.ResponseWriter, r *http.Request) {
	var req V1UploadMultipartRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("error decoding upload multipart file request", "error", err)
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

	uploadSession, partSize, requestErr := h.fileService.RequestUploadMultipartFile(r.Context(), req.FileName, req.ContentType, req.SizeBytes, req.ChecksumSha256, req.Tags)

	switch {
	case errors.Is(requestErr, domain.ErrInvalidFileType), errors.Is(requestErr, domain.ErrFileSizeTooSmall), errors.Is(requestErr, domain.ErrFileSizeTooBig), errors.Is(requestErr, domain.ErrTagNotFound):
		h.logger.Error("invalid request", "error", requestErr)
		http.Error(w, requestErr.Error(), http.StatusBadRequest)
		return
	case requestErr != nil:
		h.logger.Error("error requesting presigned url", "error", requestErr)
		http.Error(w, "internal server error", http.StatusServiceUnavailable)
		return
	case uploadSession == nil:
		http.Error(w, "upload session is nil", http.StatusInternalServerError)
		return
	default:

		resp := V1UploadMultipartResponse{
			SessionID: *uploadSession,
			PartSize:  partSize,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			h.logger.Error("error encoding response", "error", err)
		}
		return
	}

}
