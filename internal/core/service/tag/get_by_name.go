package tag

import (
	"context"
	"score-play/internal/core/domain"
)

func (t *tagService) GetTagByName(ctx context.Context, name string) (*domain.Tag, error) {
	return t.repo.FindByName(ctx, name)
}
