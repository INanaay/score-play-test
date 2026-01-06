package tag

import (
	"encoding/json"
	"net/http"
	"score-play/internal/core/domain"
	"strconv"
)

type V1ListTagsResponse struct {
	Tags       []domain.Tag `json:"tags"`
	NextMarker *string      `json:"nextMarker,omitempty"`
}

func (h *HandlerV1) ListTagsV1(w http.ResponseWriter, r *http.Request) {

	limit := r.URL.Query().Get("limit")

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if limitInt <= 0 {
		http.Error(w, "limit must be greater than zero", http.StatusBadRequest)
		return
	}

	var markerPtr *string
	if marker := r.URL.Query().Get("marker"); marker != "" {
		markerPtr = &marker
	}
	tags, nextMarker, err := h.tagService.ListTags(r.Context(), limitInt, markerPtr)
	switch {
	case err != nil:
		h.logger.Error("error listing tags", "error", err)
		http.Error(w, "internal server error", http.StatusServiceUnavailable)
		return
	default:
		resp := V1ListTagsResponse{
			Tags:       tags,
			NextMarker: nextMarker,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			h.logger.Error("error encoding response", "error", err)
		}
		return
	}

}
