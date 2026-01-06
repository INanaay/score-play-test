package postgres_test

import (
	"context"
	"score-play/internal/adapters/repository/postgres"
	"score-play/internal/core/domain"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSqlFileTagRepository(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()

	fileTagRepo := postgres.NewFileTagRepository(dbConnection)
	fileRepo := postgres.NewSqlFileRepository(dbConnection)
	tagRepo := postgres.NewSqlTagRepository(dbConnection)

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

	setupTestTags := func(t *testing.T, names ...string) map[string]uuid.UUID {
		_, err := tagRepo.CreateMany(ctx, names)
		require.NoError(t, err)

		tagMap, err := tagRepo.FindByNames(ctx, names)
		require.NoError(t, err)
		return tagMap
	}

	t.Run("Create - Nominal case", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)
		mapTags := setupTestTags(t, "test-tag")
		tagID := mapTags["test-tag"]

		err := fileTagRepo.Create(ctx, fileID, tagID)

		require.NoError(t, err)
		tags, err := fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Len(t, tags, 1)
		require.Equal(t, fileID, tags[0].FileID)
		require.Equal(t, tagID, tags[0].TagID)
	})

	t.Run("Create - Error if file does not exist", func(t *testing.T) {
		truncate()
		tags := setupTestTags(t, "test-tag")
		tagID := tags["test-tag"]

		nonExistentFileID := uuid.New()

		err := fileTagRepo.Create(ctx, nonExistentFileID, tagID)

		require.Error(t, err)
	})

	t.Run("Create - Error if tag does not exist", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		nonExistentTagID := uuid.New()

		err := fileTagRepo.Create(ctx, fileID, nonExistentTagID)

		require.Error(t, err)
	})

	t.Run("Create - Duplicate file-tag pair", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)
		tags := setupTestTags(t, "test-tag")
		tagID := tags["test-tag"]

		err := fileTagRepo.Create(ctx, fileID, tagID)
		require.NoError(t, err)

		err = fileTagRepo.Create(ctx, fileID, tagID)

		require.Error(t, err)
	})

	t.Run("FindByFileID - Multiple tags", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		mapTags := setupTestTags(t, "tag1", "tag2", "tag3")
		tag1ID := mapTags["tag1"]
		tag2ID := mapTags["tag2"]
		tag3ID := mapTags["tag3"]

		_ = fileTagRepo.Create(ctx, fileID, tag1ID)
		_ = fileTagRepo.Create(ctx, fileID, tag2ID)
		_ = fileTagRepo.Create(ctx, fileID, tag3ID)

		tags, err := fileTagRepo.FindByFileID(ctx, fileID)

		require.NoError(t, err)
		require.Len(t, tags, 3)
	})

	t.Run("FindByFileID - No tags", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		tags, err := fileTagRepo.FindByFileID(ctx, fileID)

		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("CreateMany - Nominal case", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		mapTags := setupTestTags(t, "tag1", "tag2", "tag3")
		tag1ID := mapTags["tag1"]
		tag2ID := mapTags["tag2"]
		tag3ID := mapTags["tag3"]

		tagIDs := []uuid.UUID{tag1ID, tag2ID, tag3ID}

		count, err := fileTagRepo.CreateMany(ctx, fileID, tagIDs)

		require.NoError(t, err)
		require.Equal(t, 3, count)

		tags, err := fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Len(t, tags, 3)
	})

	t.Run("CreateMany - With duplicates in input", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		mapTags := setupTestTags(t, "tag1")
		tagID := mapTags["tag1"]

		tagIDs := []uuid.UUID{tagID, tagID, tagID}

		count, err := fileTagRepo.CreateMany(ctx, fileID, tagIDs)

		require.NoError(t, err)
		require.Equal(t, 1, count)

		tags, err := fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Len(t, tags, 1)
	})

	t.Run("CreateMany - Some already exist", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		mapTags := setupTestTags(t, "tag1", "tag2", "tag3")
		tag1ID := mapTags["tag1"]
		tag2ID := mapTags["tag2"]
		tag3ID := mapTags["tag3"]

		_ = fileTagRepo.Create(ctx, fileID, tag1ID)

		tagIDs := []uuid.UUID{tag1ID, tag2ID, tag3ID}

		count, err := fileTagRepo.CreateMany(ctx, fileID, tagIDs)

		require.NoError(t, err)
		require.Equal(t, 2, count)

		tags, err := fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Len(t, tags, 3)
	})

	t.Run("CreateMany - Empty list", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		count, err := fileTagRepo.CreateMany(ctx, fileID, []uuid.UUID{})

		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("CreateMany - All already exist", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		mapTags := setupTestTags(t, "tag1", "tag2")
		tag1ID := mapTags["tag1"]
		tag2ID := mapTags["tag2"]

		_ = fileTagRepo.Create(ctx, fileID, tag1ID)
		_ = fileTagRepo.Create(ctx, fileID, tag2ID)

		tagIDs := []uuid.UUID{tag1ID, tag2ID}

		count, err := fileTagRepo.CreateMany(ctx, fileID, tagIDs)

		require.NoError(t, err)
		require.Equal(t, 0, count)

		tags, err := fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Len(t, tags, 2)
	})

	t.Run("CreateMany - Error if file does not exist", func(t *testing.T) {
		truncate()
		tags := setupTestTags(t, "tag1")
		tagID := tags["tag1"]

		nonExistentFileID := uuid.New()
		tagIDs := []uuid.UUID{tagID}

		count, err := fileTagRepo.CreateMany(ctx, nonExistentFileID, tagIDs)

		require.Error(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("CreateMany - Error if one tag does not exist", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		tags := setupTestTags(t, "tag1")
		tag1ID := tags["tag1"]

		nonExistentTagID := uuid.New()
		tagIDs := []uuid.UUID{tag1ID, nonExistentTagID}

		count, err := fileTagRepo.CreateMany(ctx, fileID, tagIDs)

		require.Error(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("DeleteByFileID - Nominal case", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		mapTags := setupTestTags(t, "tag1", "tag2", "tag3")
		tag1ID := mapTags["tag1"]
		tag2ID := mapTags["tag2"]
		tag3ID := mapTags["tag3"]

		_ = fileTagRepo.Create(ctx, fileID, tag1ID)
		_ = fileTagRepo.Create(ctx, fileID, tag2ID)
		_ = fileTagRepo.Create(ctx, fileID, tag3ID)

		tags, err := fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Len(t, tags, 3)

		err = fileTagRepo.DeleteByFileID(ctx, fileID)

		require.NoError(t, err)
		tags, err = fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("DeleteByFileID - File with no tags", func(t *testing.T) {
		truncate()
		fileID := uuid.New()
		setupTestFile(t, fileID)

		err := fileTagRepo.DeleteByFileID(ctx, fileID)

		require.NoError(t, err)
		tags, err := fileTagRepo.FindByFileID(ctx, fileID)
		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("DeleteByFileID - Non-existent file", func(t *testing.T) {
		truncate()
		nonExistentFileID := uuid.New()

		err := fileTagRepo.DeleteByFileID(ctx, nonExistentFileID)

		require.NoError(t, err)
	})

	t.Run("DeleteByFileID - Does not affect other files", func(t *testing.T) {
		truncate()
		fileID1 := uuid.New()
		fileID2 := uuid.New()
		setupTestFile(t, fileID1)
		setupTestFile(t, fileID2)

		mapTags := setupTestTags(t, "tag1", "tag2")
		tag1ID := mapTags["tag1"]
		tag2ID := mapTags["tag2"]

		_ = fileTagRepo.Create(ctx, fileID1, tag1ID)
		_ = fileTagRepo.Create(ctx, fileID1, tag2ID)
		_ = fileTagRepo.Create(ctx, fileID2, tag1ID)

		err := fileTagRepo.DeleteByFileID(ctx, fileID1)

		require.NoError(t, err)

		tags1, err := fileTagRepo.FindByFileID(ctx, fileID1)
		require.NoError(t, err)
		require.Empty(t, tags1)

		tags2, err := fileTagRepo.FindByFileID(ctx, fileID2)
		require.NoError(t, err)
		require.Len(t, tags2, 1)
		require.Equal(t, tag1ID, tags2[0].TagID)
	})
}
