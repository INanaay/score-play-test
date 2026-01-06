package tag

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"score-play/internal/core/domain"
	"unicode"
)

// V1CreateTagsRequest is the body request for Create Tags
type V1CreateTagsRequest struct {
	Tags []string `json:"tags"`
}

// CreateTagsV1 is the handler for create tags v1
func (h *HandlerV1) CreateTagsV1(w http.ResponseWriter, r *http.Request) {

	var req V1CreateTagsRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("error decoding create tags request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Tags == nil || len(req.Tags) == 0 {
		http.Error(w, "tags required", http.StatusBadRequest)
		return
	}

	for _, tag := range req.Tags {
		if tag == "" {
			http.Error(w, "tag cannot be empty", http.StatusBadRequest)
			return
		}

		for _, char := range tag {
			if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
				http.Error(w, fmt.Sprintf("tag :%s contains invalid characters", tag), http.StatusBadRequest)
				return
			}
		}

	}

	err = h.tagService.CreateTags(r.Context(), req.Tags)
	switch {
	case errors.Is(err, domain.ErrAlreadyExists):
		h.logger.Error("all tags already exist", "error", err)
		http.Error(w, "all tags already exist", http.StatusConflict)
	case err != nil:
		h.logger.Error("error creating tags", "error", err)
		http.Error(w, "internal server error", http.StatusServiceUnavailable)
		return
	default:
		w.WriteHeader(http.StatusCreated)
	}

}
