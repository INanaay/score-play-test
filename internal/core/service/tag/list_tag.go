package tag

import (
	"context"
	"score-play/internal/core/domain"
)

func (t *tagService) ListTags(ctx context.Context, limit int, marker *string) ([]domain.Tag, *string, error) {

	list, nextMarker, err := t.repo.List(ctx, limit, marker)
	if err != nil {
		return nil, nil, err
	}

	return list, nextMarker, nil
}
