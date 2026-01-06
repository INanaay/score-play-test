package file

import (
	"context"
	"fmt"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"time"

	"github.com/google/uuid"
)

func (f *fileService) RequestUploadFile(ctx context.Context, fileName string, contentType string, sizeBytes int64, checksumSha256 string, tags []string) (*uuid.UUID, *string, map[string]string, *time.Time, error) {

	if sizeBytes > f.fileUploadCfg.SingleUploadMaxSize+1 {
		return nil, nil, nil, nil, domain.ErrFileSizeTooBig
	}

	fileType, mimeType, err := f.validateMediaFile(fileName, contentType)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("%w: %w", domain.ErrInvalidFileType, err)
	}

	fileID := uuid.New()

	var presignedURL string
	var headers map[string]string
	var expiresAt *time.Time

	storageKey := fmt.Sprintf("%s/%s", fileType, fileID.String())

	txErr := f.uow.Execute(ctx, func(uow port.UnitOfWork) error {

		createErr := uow.FileRepo().Create(ctx, fileID, fileName, mimeType, fileType, sizeBytes, domain.FileStatusUploading, checksumSha256, storageKey)
		if createErr != nil {
			return createErr
		}

		validated, err := f.validateAndGetTagIDs(ctx, uow, tags)
		if err != nil {
			return err
		}

		_, err = uow.FileTagRepo().CreateMany(ctx, fileID, validated)
		if err != nil {
			return err
		}

		var storeErr error
		presignedURL, headers, expiresAt, storeErr = f.fileStorage.GeneratePresignedURLSimpleUpload(ctx, storageKey, checksumSha256)
		if storeErr != nil {
			return storeErr
		}

		return nil

	})

	if txErr != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not generate simple upload presigned url: %w", txErr)
	}
	return &fileID, &presignedURL, headers, expiresAt, nil

}
