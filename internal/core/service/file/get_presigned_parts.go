package file

import (
	"context"
	"score-play/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

// GetPresignedParts returns presigned URL for each part
func (f *fileService) GetPresignedParts(ctx context.Context, sessionID uuid.UUID, parts []domain.UploadPart) ([]domain.UploadPart, error) {

	uploadParts := make([]domain.UploadPart, 0, len(parts))

	session, err := f.uow.UploadSessionRepo().FindByIDAndActive(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	err = f.uow.UploadSessionRepo().UpdateExpiresAt(ctx, sessionID, time.Now().Add(f.fileUploadCfg.SessionTTL))
	if err != nil {
		return nil, err
	}

	fileMetadata, err := f.uow.FileRepo().FindById(ctx, session.FileID)
	if err != nil {
		return nil, err
	}

	//TODO routines
	for _, part := range parts {
		presignedPartURL, headers, expiresAt, err := f.fileStorage.GeneratePresignedURLForPart(ctx, fileMetadata.StorageKey, part.PartNumber, session.ProviderUploadID, fileMetadata.MimeType, part.ContentLength, part.ChecksumSHA256)
		if err != nil {
			return nil, err
		}
		uploadParts = append(uploadParts, domain.UploadPart{
			PartNumber:   part.PartNumber,
			Headers:      headers,
			ExpiresAt:    expiresAt,
			PresignedURL: presignedPartURL,
		})
	}

	return uploadParts, nil
}
