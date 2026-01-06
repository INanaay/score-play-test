package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"score-play/internal/core/domain"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RequestParts is a struct to represent a part in a request
type RequestParts struct {
	PartNumber    int    `json:"part_number"`
	Checksum      string `json:"checksum"`
	ContentLength int64  `json:"content_length"`
}

// ResponseParts is a struct to represent a part in response
type ResponseParts struct {
	PartNumber   int               `json:"part_number"`
	PresignedURL string            `json:"presigned_url"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Headers      map[string]string `json:"headers"`
}

type V1RetrievePresignedPartsRequest struct {
	Parts []RequestParts `json:"parts"`
}

type V1RetrievePresignedPartsResponse struct {
	Parts []ResponseParts `json:"presigned_parts"`
}

func (h *HandlerV1) RetrievePresignedPartsV1(w http.ResponseWriter, r *http.Request) {

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

	var req V1RetrievePresignedPartsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("error decoding retrieve presigned parts request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Parts == nil || len(req.Parts) == 0 {
		http.Error(w, "Request contains no parts", http.StatusBadRequest)
		return
	}

	var domainParts = make([]domain.UploadPart, 0, len(req.Parts))

	for _, part := range req.Parts {
		if part.PartNumber <= 0 {
			http.Error(w, fmt.Sprintf("part %d : invalid part number", part.PartNumber), http.StatusBadRequest)
			return
		}
		if part.ContentLength <= 0 {
			http.Error(w, fmt.Sprintf("part %d : invalid content length", part.PartNumber), http.StatusBadRequest)
			return
		}
		if part.Checksum == "" {
			http.Error(w, fmt.Sprintf("part %d : invalid checksum", part.PartNumber), http.StatusBadRequest)
			return
		}

		domainParts = append(domainParts, domain.UploadPart{
			PartNumber:     part.PartNumber,
			ChecksumSHA256: part.Checksum,
			ContentLength:  part.ContentLength,
		})
	}

	presignedParts, err := h.fileService.GetPresignedParts(r.Context(), uuidSession, domainParts)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			http.Error(w, "Session not found", http.StatusForbidden)
		}
		if errors.Is(err, domain.ErrFileMetadataNotFound) {
			http.Error(w, "no session found", http.StatusNotFound)
			return
		}
		h.logger.Error("error getting presigned parts", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var partsResponse []ResponseParts

	for _, part := range presignedParts {
		partsResponse = append(partsResponse, ResponseParts{
			PartNumber:   part.PartNumber,
			PresignedURL: part.PresignedURL,
			ExpiresAt:    *part.ExpiresAt,
			Headers:      part.Headers,
		})
	}

	response := V1RetrievePresignedPartsResponse{
		Parts: partsResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("error encoding response", "error", err)
	}
	return

}
