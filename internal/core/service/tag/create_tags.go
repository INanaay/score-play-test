package tag

import (
	"context"
)

// CreateTags creates tags by batch
func (t *tagService) CreateTags(ctx context.Context, tags []string) error {

	_, err := t.repo.CreateMany(ctx, tags)
	if err != nil {
		return err
	}

	return nil
}
