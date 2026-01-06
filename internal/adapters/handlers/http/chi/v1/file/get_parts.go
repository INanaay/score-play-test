package file

import (
	"encoding/json"
	"errors"
	"net/http"
	"score-play/internal/core/domain"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ListedPart struct {
	ETag       string `json:"etag"`
	PartNumber int    `json:"part_number"`
}

type V1GetPartsResponse struct {
	Parts       []ListedPart `json:"parts"`
	PartsMarker int          `json:"parts_marker"`
}

func (h *HandlerV1) GetPartsV1(w http.ResponseWriter, r *http.Request) {

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

	nbPartsStr := r.URL.Query().Get("nb_parts")
	markerStr := r.URL.Query().Get("marker")

	nbParts, err := strconv.Atoi(nbPartsStr)
	if err != nil {
		http.Error(w, "nb_parts must be an integer", http.StatusBadRequest)
		return
	}
	if nbParts <= 0 {
		http.Error(w, "nb_parts must be a positive integer", http.StatusBadRequest)
	}

	marker := 0
	if markerStr != "" {
		marker, err = strconv.Atoi(markerStr)
		if err != nil {
			http.Error(w, "marker must be an integer", http.StatusBadRequest)
		}
	}

	parts, marker, reqErr := h.fileService.ListParts(r.Context(), uuidSession, nbParts, marker)
	switch {
	case errors.Is(reqErr, domain.ErrSessionNotFound):
		http.Error(w, "Session not found", http.StatusForbidden)
		return
	case errors.Is(reqErr, domain.ErrFileMetadataNotFound):
		http.Error(w, "File metadata not found", http.StatusNotFound)
		return
	case reqErr != nil:
		h.logger.Error("error listing parts", "error", reqErr)
		http.Error(w, reqErr.Error(), http.StatusInternalServerError)
		return
	default:

		listedParts := make([]ListedPart, 0, len(parts))
		for _, part := range parts {
			listedParts = append(listedParts, ListedPart{
				ETag:       part.ETag,
				PartNumber: part.PartNumber,
			})
		}

		resp := V1GetPartsResponse{
			Parts:       listedParts,
			PartsMarker: marker,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			h.logger.Error("error encoding response", "error", err)
		}
		return
	}
}
