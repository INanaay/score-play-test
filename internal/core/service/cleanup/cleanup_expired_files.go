package cleanup

import (
	"context"
	"errors"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"time"
)

func (c *cleanupService) CleanupExpiredFiles(ctx context.Context, now time.Time) error {

	files, err := c.uow.FileRepo().FindExpired(ctx, now)
	if err != nil {
		return err
	}

	for _, file := range files {

		session, foundErr := c.uow.UploadSessionRepo().FindByFileID(ctx, file.ID)
		if foundErr != nil {
			if !errors.Is(foundErr, domain.ErrSessionNotFound) {
				return foundErr
			}
		}
		//Check if multipart : find session

		txErr := c.uow.Execute(ctx, func(uow port.UnitOfWork) error {

			var executeErr error

			executeErr = uow.FileRepo().UpdateStatus(ctx, file.ID, domain.FileStatusFailed)
			if executeErr != nil {
				return executeErr
			}

			executeErr = uow.FileRepo().Delete(ctx, file.ID)
			if executeErr != nil {
				return executeErr
			}

			executeErr = uow.FileTagRepo().DeleteByFileID(ctx, file.ID)
			if executeErr != nil {
				return executeErr
			}

			//Delete session if exists
			if session != nil {
				executeErr = uow.UploadSessionRepo().UpdateStatus(ctx, session.ID, domain.UploadSessionStatusAborted)
				if executeErr != nil {
					return executeErr
				}
				executeErr = c.fileStorage.AbortMultipartUpload(ctx, file.StorageKey, session.ProviderUploadID)
				if executeErr != nil {
					return executeErr
				}
			} else {
				executeErr = c.fileStorage.DeleteObject(ctx, file.StorageKey)
			}
			return nil
		})
		if txErr != nil {
			c.logger.Error("Failed to update expired sessions", "err", txErr)
		}
	}
	c.logger.Info("update expired data completed")
	return nil

}
