package file

import (
	"context"
	"errors"
	"score-play/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

func (f *fileService) GetFile(ctx context.Context, fileID uuid.UUID) (*string, *string, []domain.Tag, *time.Time, error) {

	metadata, err := f.uow.FileRepo().FindById(ctx, fileID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if metadata.Status == domain.FileStatusUploading {
		return nil, nil, nil, nil, domain.ErrFileNotReady
	}
	if metadata.Status == domain.FileStatusFailed {
		return nil, nil, nil, nil, domain.ErrFileUploadFailed
	}

	fileTags, err := f.uow.FileTagRepo().FindByFileID(ctx, metadata.ID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var tagsToFind []uuid.UUID
	for _, tag := range fileTags {
		tagsToFind = append(tagsToFind, tag.TagID)
	}

	tags, err := f.uow.TagRepo().FindByIDs(ctx, tagsToFind)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	download, expiresAt, err := f.fileStorage.GeneratePresignedURLForDownload(ctx, metadata.StorageKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if download == "" {
		return nil, nil, nil, nil, errors.New("no download url found")
	}

	return &download, &metadata.Filename, tags, expiresAt, nil

}
