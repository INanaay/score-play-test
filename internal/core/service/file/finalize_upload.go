package file

import (
	"context"
	"log"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
)

func (f *fileService) FinalizeUpload(ctx context.Context, metadata domain.FileMetadata, uploadErr error, eventType domain.EventType) error {
	sessionStatus := domain.UploadSessionStatusCompleted
	fileStatus := domain.FileStatusCompleted
	if uploadErr != nil {
		sessionStatus = domain.UploadSessionStatusAborted
		fileStatus = domain.FileStatusFailed
	}

	var session *domain.UploadSession
	var err error

	if eventType == domain.EventTypeMultipartUploadComplete {
		log.Printf("file id : %s", metadata.ID.String())
		session, err = f.uow.UploadSessionRepo().FindByFileID(ctx, metadata.ID)
		if err != nil {
			return err
		}
	}

	txErr := f.uow.Execute(ctx, func(uow port.UnitOfWork) error {
		if eventType == domain.EventTypeMultipartUploadComplete {
			if err := uow.UploadSessionRepo().UpdateStatusByFileID(ctx, metadata.ID, sessionStatus); err != nil {
				return err
			}
		}

		if err := uow.FileRepo().UpdateStatus(ctx, metadata.ID, fileStatus); err != nil {
			return err
		}

		if fileStatus == domain.FileStatusFailed {
			if err := uow.FileTagRepo().DeleteByFileID(ctx, metadata.ID); err != nil {
				return err
			}

			if err := uow.FileRepo().Delete(ctx, metadata.ID); err != nil {
				return err
			}

			if eventType == domain.EventTypeMultipartUploadComplete && session != nil {
				if err := f.fileStorage.AbortMultipartUpload(ctx, metadata.StorageKey, session.ProviderUploadID); err != nil {
					return err
				}
			} else if eventType == domain.EventTypeSimpleUploadComplete {
				if err := f.fileStorage.DeleteObject(ctx, metadata.StorageKey); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return txErr
}
