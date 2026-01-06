package file

import (
	"context"
	"score-play/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

func (f *fileService) ListParts(ctx context.Context, sessionID uuid.UUID, maxParts int, partNumberMarker int) ([]domain.UploadPart, int, error) {

	session, err := f.uow.UploadSessionRepo().FindByIDAndActive(ctx, sessionID)
	if err != nil {
		return nil, 0, err
	}

	fileMetadata, err := f.uow.FileRepo().FindById(ctx, session.FileID)
	if err != nil {
		return nil, 0, err
	}

	parts, newMarker, err := f.fileStorage.ListPartsPaginated(ctx, fileMetadata.StorageKey, session.ProviderUploadID, maxParts, partNumberMarker)
	if err != nil {
		return nil, 0, err
	}
	
	err = f.uow.UploadSessionRepo().UpdateExpiresAt(ctx, sessionID, time.Now().Add(f.fileUploadCfg.SessionTTL))
	if err != nil {
		return nil, 0, err
	}
	return parts, newMarker, nil

}
