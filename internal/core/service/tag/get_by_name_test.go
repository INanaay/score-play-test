package tag_test

import (
	"context"
	"score-play/internal/adapters/repository"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/tag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagService_ok(t *testing.T) {
	//Arrange
	ctx := context.Background()
	mockRepo := repository.NewMockTagRepository()
	tagService := tag.NewTagService(mockRepo)
	mockRepo.On("FindByName", ctx, "test").Return(&domain.Tag{
		Name: "test",
	}, nil)

	//Act
	res, err := tagService.GetTagByName(ctx, "test")

	//Assert
	require.NoError(t, err)
	require.Equal(t, "test", res.Name)
}

func TestTagService_ko(t *testing.T) {
	//Arrange
	ctx := context.Background()
	mockRepo := repository.NewMockTagRepository()
	tagService := tag.NewTagService(mockRepo)
	mockRepo.On("FindByName", ctx, "test").Return(&domain.Tag{
		Name: "test",
	}, assert.AnError)

	//Act
	_, err := tagService.GetTagByName(ctx, "test")

	//Assert
	require.ErrorIs(t, err, assert.AnError)
	mockRepo.AssertExpectations(t)
}
