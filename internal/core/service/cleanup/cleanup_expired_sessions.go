package cleanup

import (
	"context"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"time"
)

// CleanupExpiredSessions find sessions to be cleaned and cleans all data
func (c *cleanupService) CleanupExpiredSessions(ctx context.Context, now time.Time) error {

	sessions, err := c.uow.UploadSessionRepo().FindAllExpired(ctx, now)
	if err != nil {
		return err
	}

	for _, session := range sessions {

		metadata, foundErr := c.uow.FileRepo().FindById(ctx, session.FileID)
		if foundErr != nil {
			return foundErr
		}

		txErr := c.uow.Execute(ctx, func(uow port.UnitOfWork) error {

			executeErr := uow.FileRepo().UpdateStatus(ctx, session.FileID, domain.FileStatusFailed)
			if executeErr != nil {
				return executeErr
			}

			executeErr = uow.FileRepo().Delete(ctx, session.FileID)

			executeErr = uow.UploadSessionRepo().UpdateStatus(ctx, session.ID, domain.UploadSessionStatusAborted)
			if executeErr != nil {
				return executeErr
			}

			executeErr = uow.FileTagRepo().DeleteByFileID(ctx, session.FileID)
			if executeErr != nil {
				return executeErr
			}

			executeErr = c.fileStorage.AbortMultipartUpload(ctx, metadata.StorageKey, session.ProviderUploadID)
			if executeErr != nil {
				return executeErr
			}
			return nil

		})
		if txErr != nil {
			c.logger.Error("Failed to update expired sessions", "err", txErr)
		}
	}
	c.logger.Info("update expired sessions completed")
	return nil
}
