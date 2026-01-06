package file_test

import (
	"context"
	"errors"
	"score-play/internal/adapters/repository"
	"score-play/internal/adapters/storage"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/file"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFileService_RequestUploadMultipartFile_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	fileName := "video.mp4"
	contentType := "video/mp4"
	sizeBytes := int64(50000)
	checksum := "sha256"
	tags := []string{"football", "match"}

	tag1ID := uuid.New()
	tag2ID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"football": tag1ID,
		"match":    tag2ID,
	}

	expectedUploadID := "provider_123"

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, checksum).
		Return(expectedUploadID, nil)

	mockUow.GetFileRepoMock().
		On(
			"Create",
			ctx,
			mock.Anything,
			fileName,
			"video/mp4",
			domain.FileTypeVideo,
			sizeBytes,
			domain.FileStatusUploading,
			checksum,
			mock.Anything,
		).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(tagMap, nil)

	mockUow.GetFileTagRepoMock().
		On("CreateMany", ctx, mock.Anything, mock.Anything).
		Return(2, nil)

	mockUow.GetUploadSessionRepoMock().
		On("Create", ctx, mock.Anything).
		Return(nil)

	mockUow.
		On("Execute", ctx, mock.Anything).
		Return(nil)

	// Act
	sessionID, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			fileName,
			contentType,
			sizeBytes,
			checksum,
			tags,
		)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, sessionID)
	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockUow.GetTagRepoMock().AssertExpectations(t)
	mockUow.GetFileTagRepoMock().AssertExpectations(t)
}

func TestFileService_RequestUploadMultipartFile_Success_NoTags(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	fileName := "video.mp4"
	contentType := "video/mp4"
	sizeBytes := int64(50000)
	checksum := "sha256"
	tags := []string{}

	expectedUploadID := "provider_123"

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, checksum).
		Return(expectedUploadID, nil)

	mockUow.GetFileRepoMock().
		On(
			"Create",
			ctx,
			mock.Anything,
			fileName,
			"video/mp4",
			domain.FileTypeVideo,
			sizeBytes,
			domain.FileStatusUploading,
			checksum,
			mock.Anything,
		).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(map[string]uuid.UUID{}, nil)

	mockUow.GetFileTagRepoMock().
		On("CreateMany", ctx, mock.Anything, mock.Anything).
		Return(0, nil)

	mockUow.GetUploadSessionRepoMock().
		On("Create", ctx, mock.Anything).
		Return(nil)

	mockUow.
		On("Execute", ctx, mock.Anything).
		Return(nil)

	// Act
	sessionID, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			fileName,
			contentType,
			sizeBytes,
			checksum,
			tags,
		)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, sessionID)
}

func TestFileService_RequestUploadMultipartFile_TagNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	fileName := "video.mp4"
	contentType := "video/mp4"
	sizeBytes := int64(50000)
	checksum := "sha256"
	tags := []string{"football", "nonexistent"}

	tag1ID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"football": tag1ID,
	}

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, checksum).
		Return("upload_id", nil)

	mockUow.GetFileRepoMock().
		On(
			"Create",
			ctx,
			mock.Anything,
			fileName,
			"video/mp4",
			domain.FileTypeVideo,
			sizeBytes,
			domain.FileStatusUploading,
			checksum,
			mock.Anything,
		).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(tagMap, nil)

	mockUow.
		On("Execute", ctx, mock.Anything).
		Return(domain.ErrTagNotFound)

	// Act
	sessionID, partSize, err :=
		service.RequestUploadMultipartFile(
			ctx,
			fileName,
			contentType,
			sizeBytes,
			checksum,
			tags,
		)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTagNotFound)
	assert.Nil(t, sessionID)
	assert.Equal(t, 0, partSize)
}

func TestFileService_RequestUploadMultipartFile_InvalidMediaFile(t *testing.T) {
	// Arrange
	service := file.NewFileService(
		repository.NewMockUnitOfWork(),
		storage.NewMockStorage(),
		defaultCfg,
	)

	// Act
	sid, partSize, err :=
		service.RequestUploadMultipartFile(
			context.Background(),
			"doc.pdf",
			"application/pdf",
			50000,
			"abc",
			[]string{},
		)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidFileType)
	assert.Nil(t, sid)
	assert.Equal(t, 0, partSize)
}

func TestFileService_RequestUploadMultipartFile_StorageInitFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	storageErr := errors.New("fileStorage down")

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, "sha").
		Return("", storageErr)

	mockUow.
		On("Execute", ctx, mock.Anything).
		Return(storageErr)

	// Act
	_, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			"video.mp4",
			"video/mp4",
			100000,
			"sha",
			[]string{},
		)

	// Assert
	assert.Error(t, err)
	assert.ErrorContains(t, err, "could not start multipart upload")
	assert.ErrorIs(t, err, storageErr)
}

func TestFileService_RequestUploadMultipartFile_FileRepoFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	repoErr := errors.New("db error")

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, "sha").
		Return("upload_id", nil)

	mockUow.GetFileRepoMock().
		On("Create", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).
		Return(repoErr)

	mockUow.
		On("Execute", ctx, mock.Anything).Return(repoErr)

	// Act
	_, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			"video.mp4",
			"video/mp4",
			100000,
			"sha",
			[]string{},
		)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}

func TestFileService_RequestUploadMultipartFile_FindByNamesFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	tags := []string{"football"}
	findErr := errors.New("db error finding tags")

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, "sha").
		Return("upload_id", nil)

	mockUow.GetFileRepoMock().
		On("Create", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(map[string]uuid.UUID{}, findErr)

	mockUow.
		On("Execute", ctx, mock.Anything).Return(findErr)

	// Act
	_, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			"video.mp4",
			"video/mp4",
			100000,
			"sha",
			tags,
		)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, findErr)
}

func TestFileService_RequestUploadMultipartFile_CreateManyFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	tags := []string{"football"}
	tagID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"football": tagID,
	}
	createManyErr := errors.New("db error creating file tags")

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, "sha").
		Return("upload_id", nil)

	mockUow.GetFileRepoMock().
		On("Create", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(tagMap, nil)

	mockUow.GetFileTagRepoMock().
		On("CreateMany", ctx, mock.Anything, mock.Anything).
		Return(0, createManyErr)

	mockUow.
		On("Execute", ctx, mock.Anything).Return(createManyErr)

	// Act
	_, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			"video.mp4",
			"video/mp4",
			100000,
			"sha",
			tags,
		)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, createManyErr)
}

func TestFileService_RequestUploadMultipartFile_UploadSessionRepoFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	tags := []string{"football"}
	tagID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"football": tagID,
	}
	sessionErr := errors.New("session error")

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, "sha").
		Return("upload_id", nil)

	mockUow.GetFileRepoMock().
		On("Create", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(tagMap, nil)

	mockUow.GetFileTagRepoMock().
		On("CreateMany", ctx, mock.Anything, mock.Anything).
		Return(1, nil)

	mockUow.GetUploadSessionRepoMock().
		On("Create", ctx, mock.Anything).
		Return(sessionErr)

	mockUow.
		On("Execute", ctx, mock.Anything).Return(sessionErr)

	// Act
	sid, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			"video.mp4",
			"video/mp4",
			100000,
			"sha",
			tags,
		)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, sessionErr)
	assert.Nil(t, sid)
}

func TestFileService_RequestUploadMultipartFile_StorageKeyLogic(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	tags := []string{"photo"}
	tagID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"photo": tagID,
	}

	mockStorage.
		On("InitMultipartUpload", ctx, mock.Anything, "sha").
		Return("upload_id", nil)

	mockUow.GetFileRepoMock().
		On("Create", ctx, mock.Anything,
			"photo.jpg",
			"image/jpeg",
			domain.FileTypeImage,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(tagMap, nil)

	mockUow.GetFileTagRepoMock().
		On("CreateMany", ctx, mock.Anything, mock.Anything).
		Return(1, nil)

	mockUow.GetUploadSessionRepoMock().
		On("Create", ctx, mock.Anything).
		Return(nil)

	mockUow.
		On("Execute", ctx, mock.Anything).
		Return(nil)

	// Act
	_, _, err :=
		service.RequestUploadMultipartFile(
			ctx,
			"photo.jpg",
			"image/jpeg",
			1000000,
			"sha",
			tags,
		)

	// Assert
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}
