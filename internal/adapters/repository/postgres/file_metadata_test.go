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

func TestSqlFileRepository(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := postgres.NewSqlFileRepository(dbConnection)

	t.Run("Create - Success", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()

		// Act
		err := repo.Create(ctx, fileID, "test.mp4", "video/mp4", domain.FileTypeVideo, 1024, domain.FileStatusUploading, "sum", "key")

		// Assert
		require.NoError(t, err)
		file, err := repo.FindById(ctx, fileID)
		require.NoError(t, err)
		require.Equal(t, fileID, file.ID)
		require.Equal(t, "test.mp4", file.Filename)
	})

	t.Run("UpdateStatus - Success", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()
		_ = repo.Create(ctx, fileID, "test.mp4", "video/mp4", domain.FileTypeVideo, 1024, domain.FileStatusUploading, "sum", "key")

		// Act
		err := repo.UpdateStatus(ctx, fileID, domain.FileStatusCompleted)

		// Assert
		require.NoError(t, err)
		file, _ := repo.FindById(ctx, fileID)
		require.Equal(t, domain.FileStatusCompleted, file.Status)
	})

	t.Run("UpdateStatus - Not Found", func(t *testing.T) {
		// Arrange
		truncate()

		// Act
		err := repo.UpdateStatus(ctx, uuid.New(), domain.FileStatusCompleted)

		// Assert
		require.ErrorIs(t, err, domain.ErrFileMetadataNotFound)
	})

	t.Run("Delete (Soft Delete) - Success", func(t *testing.T) {
		// Arrange
		truncate()
		fileID := uuid.New()
		_ = repo.Create(ctx, fileID, "test.mp4", "video/mp4", domain.FileTypeVideo, 1024, domain.FileStatusUploading, "sum", "key")

		// Act
		err := repo.Delete(ctx, fileID)

		// Assert
		require.NoError(t, err)
		_, err = repo.FindById(ctx, fileID)
		require.ErrorIs(t, err, domain.ErrFileMetadataNotFound)
	})

	t.Run("FindById - Not Found", func(t *testing.T) {
		// Arrange
		truncate()

		// Act
		file, err := repo.FindById(ctx, uuid.New())

		// Assert
		require.Nil(t, file)
		require.ErrorIs(t, err, domain.ErrFileMetadataNotFound)
	})

	t.Run("FindExpired - Success", func(t *testing.T) {
		// Arrange
		truncate()
		expiredID := uuid.New()
		recentID := uuid.New()

		_ = repo.Create(ctx, expiredID, "old.mp4", "video/mp4", domain.FileTypeVideo, 100, domain.FileStatusUploading, "sum1", "key1")
		_ = repo.Create(ctx, recentID, "new.mp4", "video/mp4", domain.FileTypeVideo, 100, domain.FileStatusUploading, "sum2", "key2")

		// Act
		files, err := repo.FindExpired(ctx, time.Now().Add(time.Minute))

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, files)

		var found bool
		for _, f := range files {
			if f.ID == expiredID {
				found = true
			}
		}
		require.True(t, found)
	})
}
