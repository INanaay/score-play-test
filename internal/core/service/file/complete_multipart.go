package file

import (
	"context"
	"score-play/internal/core/domain"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CompleteMultipartUpload completes multipart upload
func (f *fileService) CompleteMultipartUpload(ctx context.Context, sessionID uuid.UUID, parts []domain.UploadPart) (*uuid.UUID, error) {

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

	exp := make(map[int]string, len(parts))
	for _, p := range parts {
		if _, dup := exp[p.PartNumber]; dup {
			return nil, domain.ErrDuplicatePart
		}
		exp[p.PartNumber] = strings.Trim(p.ETag, "\"")
	}

	marker := 0
	listed := 0
	for {
		parts, next, err := f.fileStorage.ListPartsPaginated(ctx, fileMetadata.StorageKey, session.ProviderUploadID, 1000, marker)
		if err != nil {
			return nil, err
		}
		for _, part := range parts {
			listed++
			got := strings.Trim(part.ETag, "\"")
			want, ok := exp[part.PartNumber]
			if !ok || want != got {
				return nil, domain.ErrMismatchETag
			}
		}
		if next == 0 {
			break
		}
		marker = next
	}
	if listed != len(parts) {
		return nil, domain.ErrMismatchNBParts
	}

	if err := f.fileStorage.CompleteMultipartUpload(ctx, fileMetadata.StorageKey, session.ProviderUploadID, parts); err != nil {
		return nil, err
	}
	return &fileMetadata.ID, nil
}
