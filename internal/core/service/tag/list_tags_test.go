package tag_test

import (
	"context"
	"errors"
	"score-play/internal/adapters/repository"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/tag"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTagService_ListTags(t *testing.T) {
	ctx := context.Background()

	t.Run("nominal - first page without marker", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 20
		tags := []domain.Tag{
			{
				ID:   uuid.New(),
				Name: "apple",
			},
			{
				ID:   uuid.New(),
				Name: "banana",
			},
			{
				ID:   uuid.New(),
				Name: "cherry",
			},
		}
		nextMarker := "cherry"

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return(tags, &nextMarker, nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, nil)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, newMarker)
		require.Equal(t, "cherry", *newMarker)
		require.Equal(t, tags, resp)
		require.Len(t, resp, 3)
		mockRepo.AssertExpectations(t)
	})

	t.Run("nominal - page with marker", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 20
		marker := "cherry"
		tags := []domain.Tag{
			{
				ID:   uuid.New(),
				Name: "date",
			},
			{
				ID:   uuid.New(),
				Name: "elderberry",
			},
		}

		mockRepo.On("List", ctx, limit, &marker).Return(tags, (*string)(nil), nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, &marker)

		// Assert
		require.NoError(t, err)
		require.Nil(t, newMarker)
		require.Equal(t, tags, resp)
		require.Len(t, resp, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 20
		emptyTags := []domain.Tag{}

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return(emptyTags, (*string)(nil), nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, nil)

		// Assert
		require.NoError(t, err)
		require.Nil(t, newMarker)
		require.Empty(t, resp)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 20
		repoErr := errors.New("database connection error")

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return([]domain.Tag(nil), (*string)(nil), repoErr)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, nil)

		// Assert
		require.Error(t, err)
		require.Equal(t, repoErr, err)
		require.Nil(t, newMarker)
		require.Nil(t, resp)
		mockRepo.AssertExpectations(t)
	})

	t.Run("limit of 1", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 1
		tags := []domain.Tag{
			{
				ID:   uuid.New(),
				Name: "apple",
			},
		}
		nextMarker := "apple"

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return(tags, &nextMarker, nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, nil)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, newMarker)
		require.Equal(t, "apple", *newMarker)
		require.Len(t, resp, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("large limit", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 100
		tags := make([]domain.Tag, 50)
		for i := 0; i < 50; i++ {
			tags[i] = domain.Tag{
				ID:   uuid.New(),
				Name: "tag" + string(rune(i)),
			}
		}

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return(tags, (*string)(nil), nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, nil)

		// Assert
		require.NoError(t, err)
		require.Nil(t, newMarker)
		require.Len(t, resp, 50)
		mockRepo.AssertExpectations(t)
	})

	t.Run("marker at the end - no more results", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 20
		marker := "zebra"
		emptyTags := []domain.Tag{}

		mockRepo.On("List", ctx, limit, &marker).Return(emptyTags, (*string)(nil), nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, &marker)

		// Assert
		require.NoError(t, err)
		require.Nil(t, newMarker)
		require.Empty(t, resp)
		mockRepo.AssertExpectations(t)
	})

	t.Run("single tag result", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 20
		tags := []domain.Tag{
			{
				ID:   uuid.New(),
				Name: "lonely",
			},
		}

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return(tags, (*string)(nil), nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, nil)

		// Assert
		require.NoError(t, err)
		require.Nil(t, newMarker)
		require.Len(t, resp, 1)
		require.Equal(t, "lonely", resp[0].Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("multiple pages scenario", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 2

		// First page
		firstPageTags := []domain.Tag{
			{ID: uuid.New(), Name: "apple"},
			{ID: uuid.New(), Name: "banana"},
		}
		firstMarker := "banana"

		// Second page
		secondPageTags := []domain.Tag{
			{ID: uuid.New(), Name: "cherry"},
			{ID: uuid.New(), Name: "date"},
		}
		secondMarker := "date"

		// Third page
		thirdPageTags := []domain.Tag{
			{ID: uuid.New(), Name: "elderberry"},
		}

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return(firstPageTags, &firstMarker, nil).Once()
		mockRepo.On("List", ctx, limit, &firstMarker).Return(secondPageTags, &secondMarker, nil).Once()
		mockRepo.On("List", ctx, limit, &secondMarker).Return(thirdPageTags, (*string)(nil), nil).Once()

		// Act - First page
		resp1, marker1, err1 := tagService.ListTags(ctx, limit, nil)
		require.NoError(t, err1)
		require.NotNil(t, marker1)
		require.Len(t, resp1, 2)

		// Act - Second page
		resp2, marker2, err2 := tagService.ListTags(ctx, limit, marker1)
		require.NoError(t, err2)
		require.NotNil(t, marker2)
		require.Len(t, resp2, 2)

		// Act - Third page (last)
		resp3, marker3, err3 := tagService.ListTags(ctx, limit, marker2)
		require.NoError(t, err3)
		require.Nil(t, marker3)
		require.Len(t, resp3, 1)

		mockRepo.AssertExpectations(t)
	})

	t.Run("nil tags returned from repo", func(t *testing.T) {
		// Arrange
		mockRepo := repository.NewMockTagRepository()
		tagService := tag.NewTagService(mockRepo)

		limit := 20

		mockRepo.On("List", ctx, limit, (*string)(nil)).Return([]domain.Tag(nil), (*string)(nil), nil)

		// Act
		resp, newMarker, err := tagService.ListTags(ctx, limit, nil)

		// Assert
		require.NoError(t, err)
		require.Nil(t, newMarker)
		require.Nil(t, resp)
		mockRepo.AssertExpectations(t)
	})

}
