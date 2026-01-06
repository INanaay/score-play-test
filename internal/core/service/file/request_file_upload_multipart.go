package file

import (
	"context"
	"fmt"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"time"

	"github.com/google/uuid"
)

func (f *fileService) RequestUploadMultipartFile(ctx context.Context, fileName string, contentType string, sizeBytes int64, checksumSha256 string, tags []string) (*uuid.UUID, int, error) {

	if sizeBytes <= f.fileUploadCfg.SingleUploadMaxSize {
		return nil, 0, domain.ErrFileSizeTooSmall
	}

	if sizeBytes > f.fileUploadCfg.MultipartUploadMaxSize {
		return nil, 0, domain.ErrFileSizeTooBig
	}

	fileType, mimeType, err := f.validateMediaFile(fileName, contentType)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %w", domain.ErrInvalidFileType, err)
	}

	fileID := uuid.New()
	storageKey := fmt.Sprintf("%s/%s", fileType, fileID.String())
	uploadSessionID := uuid.New()
	uploadID := ""

	txErr := f.uow.Execute(ctx, func(uow port.UnitOfWork) error {

		var storeErr error
		uploadID, storeErr = f.fileStorage.InitMultipartUpload(ctx, storageKey, checksumSha256)
		if storeErr != nil {
			return storeErr
		}

		metadataErr := uow.FileRepo().Create(ctx, fileID, fileName, mimeType, fileType, sizeBytes, domain.FileStatusUploading, checksumSha256, storageKey)
		if metadataErr != nil {
			return metadataErr
		}

		validated, err := f.validateAndGetTagIDs(ctx, uow, tags)
		if err != nil {
			return err
		}

		_, err = uow.FileTagRepo().CreateMany(ctx, fileID, validated)
		if err != nil {
			return err
		}

		uploadSessionErr := uow.UploadSessionRepo().Create(ctx, domain.UploadSession{
			ID:               uploadSessionID,
			FileID:           fileID,
			ProviderUploadID: uploadID,
			PartSize:         f.fileUploadCfg.PartSize,
			ExpiresAt:        time.Now().Add(f.fileUploadCfg.SessionTTL),
			Status:           domain.UploadSessionStatusOpen,
		})
		if uploadSessionErr != nil {
			return uploadSessionErr
		}
		return nil
	})
	if txErr != nil {
		return nil, 0, fmt.Errorf("could not start multipart upload: %w", txErr)
	}
	return &uploadSessionID, f.fileUploadCfg.PartSize, nil
}
