package postgres_test

import (
	"context"
	"fmt"
	"score-play/internal/adapters/repository/postgres"
	"score-play/internal/core/domain"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSqlTagRepository_CreateMany(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagRepo := postgres.NewSqlTagRepository(dbConnection)

	t.Run("nominal", func(t *testing.T) {
		truncate()
		tags := []string{"test", "test2", "test3", "test4"}
		nb, err := tagRepo.CreateMany(ctx, tags)
		require.NoError(t, err)
		require.Equal(t, len(tags), nb)
	})

	t.Run("CreateMany same tag multiple times", func(t *testing.T) {
		truncate()
		tags := []string{"test", "test", "test", "test4", "test3"}
		nb, err := tagRepo.CreateMany(ctx, tags)
		require.NoError(t, err)
		require.Equal(t, 3, nb)
	})

	t.Run("CreateMany same tag multiple times case insensitive", func(t *testing.T) {
		truncate()
		tags := []string{"Test", "tEst", "TEST", "test4", "test3"}
		nb, err := tagRepo.CreateMany(ctx, tags)
		require.NoError(t, err)
		require.Equal(t, 3, nb)
	})

	t.Run("CreateMany same tag when already exists", func(t *testing.T) {
		truncate()
		_, err := tagRepo.CreateMany(ctx, []string{"TeSt"})
		require.NoError(t, err)
		tags := []string{"Test", "tEst", "TEST", "test4", "test3"}
		nb, err := tagRepo.CreateMany(ctx, tags)
		require.NoError(t, err)
		require.Equal(t, 2, nb)
	})

	t.Run("all already exist", func(t *testing.T) {
		truncate()
		_, err := tagRepo.CreateMany(ctx, []string{"TeSt"})
		require.NoError(t, err)
		_, err = tagRepo.CreateMany(ctx, []string{"TeSt2"})
		require.NoError(t, err)

		tags := []string{"Test", "tEst", "TEST2", "test2", "test2"}
		nb, err := tagRepo.CreateMany(ctx, tags)
		require.ErrorIs(t, err, domain.ErrAlreadyExists)
		require.Equal(t, 0, nb)
	})
}

func TestSqlTagRepository_FindByName(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagRepo := postgres.NewSqlTagRepository(dbConnection)

	t.Run("nominal", func(t *testing.T) {
		truncate()
		_, err := tagRepo.CreateMany(ctx, []string{"test"})
		require.NoError(t, err)

		domainTag, err := tagRepo.FindByName(ctx, "test")
		require.NoError(t, err)
		require.NotNil(t, domainTag)
		require.NotEmpty(t, domainTag.Name)
		require.Equal(t, "test", domainTag.Name)
		require.NotEmpty(t, domainTag.ID)
		require.NotEmpty(t, domainTag.CreatedAt)
	})

	t.Run("find tag case insensitive", func(t *testing.T) {
		truncate()
		_, err := tagRepo.CreateMany(ctx, []string{"MeSsI"})
		require.NoError(t, err)
		domainTag, err := tagRepo.FindByName(ctx, "Messi")
		require.NoError(t, err)
		require.NotNil(t, domainTag)
		require.Equal(t, "messi", domainTag.Name)

		domainTag, err = tagRepo.FindByName(ctx, "MESSi")
		require.NoError(t, err)
		require.NotNil(t, domainTag)
		require.Equal(t, "messi", domainTag.Name)
	})

	t.Run("Not found", func(t *testing.T) {
		truncate()
		domainTag, err := tagRepo.FindByName(ctx, "not-found")
		require.ErrorIs(t, err, domain.ErrTagNotFound)
		require.Nil(t, domainTag)
	})
}

func TestSqlTagRepository_FindByNames(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagRepo := postgres.NewSqlTagRepository(dbConnection)

	t.Run("nominal - find multiple tags", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"football", "basketball", "tennis"})
		require.NoError(t, err)

		tagMap, err := tagRepo.FindByNames(ctx, []string{"football", "basketball", "tennis"})

		require.NoError(t, err)
		require.Len(t, tagMap, 3)
		require.NotEmpty(t, tagMap["football"])
		require.NotEmpty(t, tagMap["basketball"])
		require.NotEmpty(t, tagMap["tennis"])
	})

	t.Run("case insensitive", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"MeSsI", "RoNaLdO"})
		require.NoError(t, err)

		tagMap, err := tagRepo.FindByNames(ctx, []string{"messi", "RONALDO"})

		require.NoError(t, err)
		require.Len(t, tagMap, 2)
		require.NotEmpty(t, tagMap["messi"])
		require.NotEmpty(t, tagMap["ronaldo"])
	})

	t.Run("some tags not found", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"existing"})
		require.NoError(t, err)

		tagMap, err := tagRepo.FindByNames(ctx, []string{"existing", "notfound1", "notfound2"})

		require.NoError(t, err)
		require.Len(t, tagMap, 1)
		require.NotEmpty(t, tagMap["existing"])
		require.Empty(t, tagMap["notfound1"])
		require.Empty(t, tagMap["notfound2"])
	})

	t.Run("empty input", func(t *testing.T) {
		truncate()

		tagMap, err := tagRepo.FindByNames(ctx, []string{})

		require.NoError(t, err)
		require.Empty(t, tagMap)
	})

	t.Run("nil input", func(t *testing.T) {
		truncate()

		tagMap, err := tagRepo.FindByNames(ctx, nil)

		require.NoError(t, err)
		require.Empty(t, tagMap)
	})

	t.Run("duplicates in input", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"test"})
		require.NoError(t, err)

		tagMap, err := tagRepo.FindByNames(ctx, []string{"test", "test", "TEST", "Test"})

		require.NoError(t, err)
		require.Len(t, tagMap, 1)
		require.NotEmpty(t, tagMap["test"])
	})

	t.Run("all tags not found", func(t *testing.T) {
		truncate()

		tagMap, err := tagRepo.FindByNames(ctx, []string{"notfound1", "notfound2", "notfound3"})

		require.NoError(t, err)
		require.Empty(t, tagMap)
	})

	t.Run("single tag", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"solo"})
		require.NoError(t, err)

		tagMap, err := tagRepo.FindByNames(ctx, []string{"solo"})

		require.NoError(t, err)
		require.Len(t, tagMap, 1)
		require.NotEmpty(t, tagMap["solo"])
	})

	t.Run("large batch", func(t *testing.T) {
		truncate()

		names := make([]string, 50)
		for i := 0; i < 50; i++ {
			names[i] = fmt.Sprintf("tag%d", i)
		}
		_, err := tagRepo.CreateMany(ctx, names)
		require.NoError(t, err)

		tagMap, err := tagRepo.FindByNames(ctx, names)

		require.NoError(t, err)
		require.Len(t, tagMap, 50)
		for _, name := range names {
			require.NotEmpty(t, tagMap[name])
		}
	})
}

func TestSqlTagRepository_FindByIDs(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagRepo := postgres.NewSqlTagRepository(dbConnection)

	t.Run("nominal - find multiple tags by IDs", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"football", "basketball", "tennis"})
		require.NoError(t, err)

		football, err := tagRepo.FindByName(ctx, "football")
		require.NoError(t, err)
		basketball, err := tagRepo.FindByName(ctx, "basketball")
		require.NoError(t, err)
		tennis, err := tagRepo.FindByName(ctx, "tennis")
		require.NoError(t, err)

		ids := []uuid.UUID{football.ID, basketball.ID, tennis.ID}

		tags, err := tagRepo.FindByIDs(ctx, ids)

		require.NoError(t, err)
		require.Len(t, tags, 3)
	})

	t.Run("single tag by ID", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"solo"})
		require.NoError(t, err)

		solo, err := tagRepo.FindByName(ctx, "solo")
		require.NoError(t, err)

		tags, err := tagRepo.FindByIDs(ctx, []uuid.UUID{solo.ID})

		require.NoError(t, err)
		require.Len(t, tags, 1)
	})

	t.Run("empty input", func(t *testing.T) {
		truncate()

		tags, err := tagRepo.FindByIDs(ctx, []uuid.UUID{})

		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("nil input", func(t *testing.T) {
		truncate()

		tags, err := tagRepo.FindByIDs(ctx, nil)

		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("some IDs not found", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"existing"})
		require.NoError(t, err)

		existing, err := tagRepo.FindByName(ctx, "existing")
		require.NoError(t, err)

		nonExistentID1 := uuid.New()
		nonExistentID2 := uuid.New()

		tags, err := tagRepo.FindByIDs(ctx, []uuid.UUID{existing.ID, nonExistentID1, nonExistentID2})

		require.NoError(t, err)
		require.Len(t, tags, 1)
	})

	t.Run("all IDs not found", func(t *testing.T) {
		truncate()

		nonExistentID1 := uuid.New()
		nonExistentID2 := uuid.New()
		nonExistentID3 := uuid.New()

		tags, err := tagRepo.FindByIDs(ctx, []uuid.UUID{nonExistentID1, nonExistentID2, nonExistentID3})

		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("duplicate IDs in input", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"test"})
		require.NoError(t, err)

		test, err := tagRepo.FindByName(ctx, "test")
		require.NoError(t, err)

		tags, err := tagRepo.FindByIDs(ctx, []uuid.UUID{test.ID, test.ID, test.ID})

		require.NoError(t, err)
		require.Len(t, tags, 1)
	})

	t.Run("large batch - 50 tags", func(t *testing.T) {
		truncate()

		names := make([]string, 50)
		for i := 0; i < 50; i++ {
			names[i] = fmt.Sprintf("tag%d", i)
		}
		_, err := tagRepo.CreateMany(ctx, names)
		require.NoError(t, err)

		ids := make([]uuid.UUID, 50)
		for i, name := range names {
			tag, err := tagRepo.FindByName(ctx, name)
			require.NoError(t, err)
			ids[i] = tag.ID
		}

		tags, err := tagRepo.FindByIDs(ctx, ids)

		require.NoError(t, err)
		require.Len(t, tags, 50)
	})

	t.Run("verify names are lowercase", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"MixedCase", "UPPERCASE"})
		require.NoError(t, err)

		mixedCase, err := tagRepo.FindByName(ctx, "MixedCase")
		require.NoError(t, err)
		uppercase, err := tagRepo.FindByName(ctx, "UPPERCASE")
		require.NoError(t, err)

		tags, err := tagRepo.FindByIDs(ctx, []uuid.UUID{mixedCase.ID, uppercase.ID})

		require.NoError(t, err)
		require.Len(t, tags, 2)
	})
}

func TestSqlTagRepository_List(t *testing.T) {
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagRepo := postgres.NewSqlTagRepository(dbConnection)

	t.Run("nominal - first page without marker", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "banana", "cherry", "date", "elderberry"})
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 3, nil)

		require.NoError(t, err)
		require.Len(t, result, 3)
		require.Equal(t, "apple", result[0].Name)
		require.Equal(t, "banana", result[1].Name)
		require.Equal(t, "cherry", result[2].Name)
		require.NotNil(t, nextMarker)
		require.Equal(t, "cherry", *nextMarker)
	})

	t.Run("nominal - second page with marker", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "banana", "cherry", "date", "elderberry"})
		require.NoError(t, err)

		marker := "cherry"

		result, nextMarker, err := tagRepo.List(ctx, 2, &marker)

		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "date", result[0].Name)
		require.Equal(t, "elderberry", result[1].Name)
		require.Nil(t, nextMarker)
	})

	t.Run("last page - no next marker", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "banana", "cherry"})
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 10, nil)

		require.NoError(t, err)
		require.Len(t, result, 3)
		require.Nil(t, nextMarker)
	})

	t.Run("exact limit match", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "banana", "cherry"})
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 3, nil)

		require.NoError(t, err)
		require.Len(t, result, 3)
		require.Nil(t, nextMarker)
	})

	t.Run("empty database", func(t *testing.T) {
		truncate()

		result, nextMarker, err := tagRepo.List(ctx, 10, nil)

		require.NoError(t, err)
		require.Empty(t, result)
		require.Nil(t, nextMarker)
	})

	t.Run("case insensitive sorting", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"Zebra", "apple", "BANANA", "Cherry"})
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 10, nil)

		require.NoError(t, err)
		require.Len(t, result, 4)
		require.Equal(t, "apple", result[0].Name)
		require.Equal(t, "banana", result[1].Name)
		require.Equal(t, "cherry", result[2].Name)
		require.Equal(t, "zebra", result[3].Name)
		require.Nil(t, nextMarker)
	})

	t.Run("marker case insensitive", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "banana", "cherry", "date"})
		require.NoError(t, err)

		marker := "BANANA"

		result, nextMarker, err := tagRepo.List(ctx, 10, &marker)

		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "cherry", result[0].Name)
		require.Equal(t, "date", result[1].Name)
		require.Nil(t, nextMarker)
	})

	t.Run("default limit when zero", func(t *testing.T) {
		truncate()

		var tags []string
		for i := 0; i < 25; i++ {
			tags = append(tags, fmt.Sprintf("tag%03d", i))
		}
		_, err := tagRepo.CreateMany(ctx, tags)
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 0, nil)

		require.NoError(t, err)
		require.Len(t, result, 20)
		require.NotNil(t, nextMarker)
	})

	t.Run("max limit enforcement", func(t *testing.T) {
		truncate()

		var tags []string
		for i := 0; i < 150; i++ {
			tags = append(tags, fmt.Sprintf("tag%03d", i))
		}
		_, err := tagRepo.CreateMany(ctx, tags)
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 200, nil)

		require.NoError(t, err)
		require.Len(t, result, 100)
		require.NotNil(t, nextMarker)
	})

	t.Run("negative limit", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "banana", "cherry"})
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, -5, nil)

		require.NoError(t, err)
		require.Len(t, result, 3)
		require.Nil(t, nextMarker)
	})

	t.Run("pagination through all pages", func(t *testing.T) {
		truncate()

		expectedTags := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
		_, err := tagRepo.CreateMany(ctx, expectedTags)
		require.NoError(t, err)

		var allTags []domain.Tag
		var marker *string
		pageSize := 3

		for {
			result, nextMarker, err := tagRepo.List(ctx, pageSize, marker)
			require.NoError(t, err)

			allTags = append(allTags, result...)

			if nextMarker == nil {
				break
			}
			marker = nextMarker
		}

		require.Len(t, allTags, 10)
		for i, tag := range allTags {
			require.Equal(t, expectedTags[i], tag.Name)
		}
	})

	t.Run("marker not found - returns tags after position", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "cherry", "elderberry"})
		require.NoError(t, err)

		marker := "banana"

		result, nextMarker, err := tagRepo.List(ctx, 10, &marker)

		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "cherry", result[0].Name)
		require.Equal(t, "elderberry", result[1].Name)
		require.Nil(t, nextMarker)
	})

	t.Run("single tag in database", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"lonely"})
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 10, nil)

		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, "lonely", result[0].Name)
		require.Nil(t, nextMarker)
	})

	t.Run("limit of 1", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"apple", "banana", "cherry"})
		require.NoError(t, err)

		result, nextMarker, err := tagRepo.List(ctx, 1, nil)

		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, "apple", result[0].Name)
		require.NotNil(t, nextMarker)
		require.Equal(t, "apple", *nextMarker)
	})

	t.Run("verify all fields are populated", func(t *testing.T) {
		truncate()

		_, err := tagRepo.CreateMany(ctx, []string{"complete"})
		require.NoError(t, err)

		result, _, err := tagRepo.List(ctx, 10, nil)

		require.NoError(t, err)
		require.Len(t, result, 1)
		require.NotEmpty(t, result[0].ID)
		require.NotEmpty(t, result[0].Name)
		require.NotEmpty(t, result[0].CreatedAt)
		require.Equal(t, "complete", result[0].Name)
	})
}
