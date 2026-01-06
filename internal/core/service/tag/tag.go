package tag

import "score-play/internal/core/port"

type tagService struct {
	repo port.TagRepository
}

// NewTagService creates a new tag service
func NewTagService(repo port.TagRepository) port.TagService {
	return &tagService{repo: repo}
}
