package tag_test

import (
	"context"
	"score-play/internal/adapters/repository"
	"score-play/internal/core/service/tag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTags_ok(t *testing.T) {

	//Arrange
	mockRepo := repository.NewMockTagRepository()
	tagService := tag.NewTagService(mockRepo)
	ctx := context.Background()
	tags := []string{"test1", "test2", "test3"}
	mockRepo.On("CreateMany", ctx, tags).Return(len(tags), nil)

	//Act
	err := tagService.CreateTags(ctx, tags)

	//Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCreateTags_ko(t *testing.T) {

	//Arrange
	mockRepo := repository.NewMockTagRepository()
	tagService := tag.NewTagService(mockRepo)
	ctx := context.Background()
	tags := []string{"test1", "test2", "test3"}
	mockRepo.On("CreateMany", ctx, tags).Return(len(tags), assert.AnError)

	//Act
	err := tagService.CreateTags(ctx, tags)

	//Assert
	require.ErrorIs(t, err, assert.AnError)
	mockRepo.AssertExpectations(t)
}
