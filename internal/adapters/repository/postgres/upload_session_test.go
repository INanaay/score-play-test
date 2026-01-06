package postgres_test

import (
	"context"
	"score-play/internal/adapters/repository/postgres"
	"score-play/internal/core/domain"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSqlUploadSessionRepository(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()

	sessionRepo := postgres.NewSQLUploadSessionRepository(dbConnection)
	fileRepo := postgres.NewSqlFileRepository(dbConnection)
	setupTestFile := func(t *testing.T, id uuid.UUID) {
		err := fileRepo.Create(
			ctx,
			id,
			"video.mp4",
			"video/mp4",
			domain.FileTypeVideo,
			1024*1024,
			domain.FileStatusUploading,
			"checksum-"+id.String(),
			"temp/path/"+id.String(),
		)
		require.NoError(t, err)
	}

	t.Run("Create - Nominal case", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		session := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID,
			ProviderUploadID: "aws-upload-id-999",
			PartSize:         5242880,
			ExpiresAt:        time.Now().Add(time.Hour).Round(time.Microsecond),
			Status:           domain.UploadSessionStatusOpen,
		}

		// Act
		err := sessionRepo.Create(ctx, session)

		// Assert
		require.NoError(t, err)
		saved, err := sessionRepo.FindByID(ctx, session.ID)
		require.NoError(t, err)
		require.Equal(t, session.ID, saved.ID)
		require.Equal(t, session.ProviderUploadID, saved.ProviderUploadID)
		require.WithinDuration(t, session.ExpiresAt, saved.ExpiresAt, time.Second)
	})

	t.Run("Create - Error if file does not exist", func(t *testing.T) {
		// Arrange
		truncate()
		session := domain.UploadSession{
			ID:     uuid.New(),
			FileID: uuid.New(),
			Status: domain.UploadSessionStatusOpen,
		}

		// Act
		err := sessionRepo.Create(ctx, session)

		// Assert
		require.Error(t, err)
	})

	t.Run("UpdateExpiresAt - Success", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)
		sessionID := uuid.New()
		_ = sessionRepo.Create(ctx, domain.UploadSession{
			ID: sessionID, FileID: fileID, Status: domain.UploadSessionStatusOpen, ExpiresAt: time.Now().Add(time.Hour),
		})
		newExpiry := time.Now().Add(10 * time.Hour).Round(time.Microsecond)

		// Act
		err := sessionRepo.UpdateExpiresAt(ctx, sessionID, newExpiry)

		// Assert
		require.NoError(t, err)
		updated, _ := sessionRepo.FindByID(ctx, sessionID)
		require.WithinDuration(t, newExpiry, updated.ExpiresAt, time.Second)
	})

	t.Run("FindByFileID - Nominal case", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)
		sessionID := uuid.New()

		session := domain.UploadSession{
			ID:               sessionID,
			FileID:           fileID,
			ProviderUploadID: "aws-upload-id-123",
			PartSize:         5242880,
			ExpiresAt:        time.Now().Add(time.Hour).Round(time.Microsecond),
			Status:           domain.UploadSessionStatusOpen,
		}
		err := sessionRepo.Create(ctx, session)
		require.NoError(t, err)

		// Act
		found, err := sessionRepo.FindByFileID(ctx, fileID)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, sessionID, found.ID)
		require.Equal(t, fileID, found.FileID)
		require.Equal(t, "aws-upload-id-123", found.ProviderUploadID)
		require.Equal(t, domain.UploadSessionStatusOpen, found.Status)
	})

	t.Run("FindByFileID - Session not found when no session exists", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		// Act
		found, err := sessionRepo.FindByFileID(ctx, fileID)

		// Assert
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrSessionNotFound)
		require.Nil(t, found)
	})

	t.Run("FindByFileID - Session not found when only closed sessions exist", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		completedSessionID := uuid.New()
		_ = sessionRepo.Create(ctx, domain.UploadSession{
			ID:               completedSessionID,
			FileID:           fileID,
			ProviderUploadID: "completed-upload-id",
			Status:           domain.UploadSessionStatusCompleted,
			ExpiresAt:        time.Now().Add(time.Hour),
			PartSize:         5242880,
		})

		abortedSessionID := uuid.New()
		_ = sessionRepo.Create(ctx, domain.UploadSession{
			ID:               abortedSessionID,
			FileID:           fileID,
			ProviderUploadID: "aborted-upload-id",
			Status:           domain.UploadSessionStatusAborted,
			ExpiresAt:        time.Now().Add(time.Hour),
			PartSize:         5242880,
		})

		// Act
		found, err := sessionRepo.FindByFileID(ctx, fileID)

		// Assert
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrSessionNotFound)
		require.Nil(t, found)
	})

	t.Run("FindAllExpired - Returns expired open sessions", func(t *testing.T) {
		// Arrange
		truncate()
		now := time.Now().Round(time.Microsecond)

		fileID1 := uuid.New()
		setupTestFile(t, fileID1)
		expiredSession1 := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID1,
			ProviderUploadID: "expired-1",
			PartSize:         5242880,
			ExpiresAt:        now.Add(-2 * time.Hour),
			Status:           domain.UploadSessionStatusOpen,
		}
		err := sessionRepo.Create(ctx, expiredSession1)
		require.NoError(t, err)

		fileID2 := uuid.New()
		setupTestFile(t, fileID2)
		expiredSession2 := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID2,
			ProviderUploadID: "expired-2",
			PartSize:         5242880,
			ExpiresAt:        now.Add(-1 * time.Hour),
			Status:           domain.UploadSessionStatusOpen,
		}
		err = sessionRepo.Create(ctx, expiredSession2)
		require.NoError(t, err)

		fileID3 := uuid.New()
		setupTestFile(t, fileID3)
		validSession := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID3,
			ProviderUploadID: "valid",
			PartSize:         5242880,
			ExpiresAt:        now.Add(2 * time.Hour),
			Status:           domain.UploadSessionStatusOpen,
		}
		err = sessionRepo.Create(ctx, validSession)
		require.NoError(t, err)

		fileID4 := uuid.New()
		setupTestFile(t, fileID4)
		expiredCompletedSession := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID4,
			ProviderUploadID: "expired-completed",
			PartSize:         5242880,
			ExpiresAt:        now.Add(-3 * time.Hour),
			Status:           domain.UploadSessionStatusCompleted,
		}
		err = sessionRepo.Create(ctx, expiredCompletedSession)
		require.NoError(t, err)

		// Act
		expiredSessions, err := sessionRepo.FindAllExpired(ctx, now)

		// Assert
		require.NoError(t, err)
		require.Len(t, expiredSessions, 2)

		expiredIDs := make(map[uuid.UUID]bool)
		for _, session := range expiredSessions {
			expiredIDs[session.ID] = true
			require.Equal(t, domain.UploadSessionStatusOpen, session.Status)
			require.True(t, session.ExpiresAt.Before(now))
		}

		require.True(t, expiredIDs[expiredSession1.ID])
		require.True(t, expiredIDs[expiredSession2.ID])
		require.False(t, expiredIDs[validSession.ID])
		require.False(t, expiredIDs[expiredCompletedSession.ID])
	})

	t.Run("FindAllExpired - Returns empty list when no expired sessions", func(t *testing.T) {
		// Arrange
		truncate()
		now := time.Now().Round(time.Microsecond)

		fileID := uuid.New()
		setupTestFile(t, fileID)
		validSession := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID,
			ProviderUploadID: "valid",
			PartSize:         5242880,
			ExpiresAt:        now.Add(2 * time.Hour),
			Status:           domain.UploadSessionStatusOpen,
		}
		err := sessionRepo.Create(ctx, validSession)
		require.NoError(t, err)

		// Act
		expiredSessions, err := sessionRepo.FindAllExpired(ctx, now)

		// Assert
		require.NoError(t, err)
		require.Empty(t, expiredSessions)
	})

	t.Run("FindAllExpired - Returns empty list when no sessions exist", func(t *testing.T) {
		// Arrange
		truncate()
		now := time.Now()

		// Act
		expiredSessions, err := sessionRepo.FindAllExpired(ctx, now)

		// Assert
		require.NoError(t, err)
		require.Empty(t, expiredSessions)
	})

	t.Run("FindAllExpired - Ignores aborted sessions", func(t *testing.T) {
		// Arrange
		truncate()
		now := time.Now().Round(time.Microsecond)

		fileID1 := uuid.New()
		setupTestFile(t, fileID1)
		expiredAbortedSession := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID1,
			ProviderUploadID: "expired-aborted",
			PartSize:         5242880,
			ExpiresAt:        now.Add(-2 * time.Hour),
			Status:           domain.UploadSessionStatusAborted,
		}
		err := sessionRepo.Create(ctx, expiredAbortedSession)
		require.NoError(t, err)

		fileID2 := uuid.New()
		setupTestFile(t, fileID2)
		expiredOpenSession := domain.UploadSession{
			ID:               uuid.New(),
			FileID:           fileID2,
			ProviderUploadID: "expired-open",
			PartSize:         5242880,
			ExpiresAt:        now.Add(-1 * time.Hour),
			Status:           domain.UploadSessionStatusOpen,
		}
		err = sessionRepo.Create(ctx, expiredOpenSession)
		require.NoError(t, err)

		// Act
		expiredSessions, err := sessionRepo.FindAllExpired(ctx, now)

		// Assert
		require.NoError(t, err)
		require.Len(t, expiredSessions, 1)
		require.Equal(t, expiredOpenSession.ID, expiredSessions[0].ID)
		require.Equal(t, domain.UploadSessionStatusOpen, expiredSessions[0].Status)
	})
}
