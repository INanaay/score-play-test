package file_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"score-play/internal/adapters/repository"
	"score-play/internal/adapters/storage"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/file"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFileService_RequestUploadFile_Success_VideoFile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	fileService := file.NewFileService(mockUow, mockStorage, defaultCfg)

	fileName := "video.mp4"
	contentType := "video/mp4"
	sizeBytes := int64(1000)
	checksum := "abc123"
	tags := []string{"football", "highlights"}

	tag1ID := uuid.New()
	tag2ID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"football":   tag1ID,
		"highlights": tag2ID,
	}

	presignedURL := "https://minio.example.com/bucket/video-id"
	headers := map[string]string{
		"Content-Type":                 "video/mp4",
		"Content-Length":               "1000",
		"x-amz-checksum-sha256":        checksum,
		"x-amz-sdk-checksum-algorithm": "SHA256",
	}
	expiresAt := time.Now().Add(15 * time.Minute)

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

	mockStorage.
		On(
			"GeneratePresignedURLSimpleUpload",
			ctx,
			mock.Anything,
			checksum,
		).
		Return(presignedURL, headers, &expiresAt, nil)

	mockUow.
		On("Execute", ctx, mock.Anything).
		Return(nil)

	// Act
	fileID, url, resultHeaders, resultExpiresAt, err :=
		fileService.RequestUploadFile(ctx, fileName, contentType, sizeBytes, checksum, tags)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, fileID)
	assert.NotNil(t, url)
	assert.Equal(t, presignedURL, *url)
	assert.Equal(t, headers, resultHeaders)
	assert.NotNil(t, resultExpiresAt)

	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockUow.GetFileRepoMock().AssertExpectations(t)
	mockUow.GetTagRepoMock().AssertExpectations(t)
	mockUow.GetFileTagRepoMock().AssertExpectations(t)
}

func TestFileService_RequestUploadFile_Success_ImageJPEG(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	fileService := file.NewFileService(mockUow, mockStorage, defaultCfg)

	fileName := "photo.jpg"
	contentType := "image/jpeg"
	sizeBytes := int64(1000)
	checksum := "img123"
	tags := []string{"nature"}

	tagID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"nature": tagID,
	}

	presignedURL := "https://minio.example.com/bucket/image-id"
	headers := map[string]string{
		"Content-Type":                 "image/jpeg",
		"Content-Length":               "1000",
		"x-amz-checksum-sha256":        checksum,
		"x-amz-sdk-checksum-algorithm": "SHA256",
	}
	expiresAt := time.Now().Add(15 * time.Minute)

	mockUow.GetFileRepoMock().
		On(
			"Create",
			ctx,
			mock.Anything,
			fileName,
			"image/jpeg",
			domain.FileTypeImage,
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
		Return(1, nil)

	mockStorage.
		On(
			"GeneratePresignedURLSimpleUpload",
			ctx,
			mock.Anything,
			checksum,
		).
		Return(presignedURL, headers, &expiresAt, nil)

	mockUow.On("Execute", ctx, mock.Anything).Return(nil)

	// Act
	_, url, resultHeaders, _, err :=
		fileService.RequestUploadFile(ctx, fileName, contentType, sizeBytes, checksum, tags)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, presignedURL, *url)
	assert.Equal(t, headers, resultHeaders)
}

func TestFileService_RequestUploadFile_Success_NoTags(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	fileService := file.NewFileService(mockUow, mockStorage, defaultCfg)

	fileName := "video.mp4"
	contentType := "video/mp4"
	sizeBytes := int64(1000)
	checksum := "abc123"
	tags := []string{}

	presignedURL := "https://minio.example.com/bucket/video-id"
	headers := map[string]string{
		"Content-Type": "video/mp4",
	}
	expiresAt := time.Now().Add(15 * time.Minute)

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

	mockStorage.
		On(
			"GeneratePresignedURLSimpleUpload",
			ctx,
			mock.Anything,
			checksum,
		).
		Return(presignedURL, headers, &expiresAt, nil)

	mockUow.On("Execute", ctx, mock.Anything).Return(nil)

	// Act
	fileID, url, resultHeaders, resultExpiresAt, err :=
		fileService.RequestUploadFile(ctx, fileName, contentType, sizeBytes, checksum, tags)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, fileID)
	assert.NotNil(t, url)
	assert.Equal(t, presignedURL, *url)
	assert.Equal(t, headers, resultHeaders)
	assert.NotNil(t, resultExpiresAt)
}

func TestFileService_RequestUploadFile_TagNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	fileService := file.NewFileService(mockUow, mockStorage, defaultCfg)

	fileName := "video.mp4"
	contentType := "video/mp4"
	sizeBytes := int64(1000)
	checksum := "abc123"
	tags := []string{"football", "nonexistent"}

	tag1ID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"football": tag1ID,
		// "nonexistent" is missing
	}

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

	mockUow.On("Execute", ctx, mock.Anything).Return(domain.ErrTagNotFound)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, fileName, contentType, sizeBytes, checksum, tags)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTagNotFound)
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}

func TestFileService_RequestUploadFile_InvalidContentType(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileService := file.NewFileService(
		repository.NewMockUnitOfWork(),
		storage.NewMockStorage(),
		defaultCfg,
	)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, "doc.pdf", "application/pdf", 1024, "abc", []string{})

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidFileType))
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}

func TestFileService_RequestUploadFile_MismatchedExtension(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileService := file.NewFileService(
		repository.NewMockUnitOfWork(),
		storage.NewMockStorage(),
		defaultCfg,
	)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, "video.jpg", "video/mp4", 1024, "abc", []string{})

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidFileType))
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}

func TestFileService_RequestUploadFile_FileTooBig(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileService := file.NewFileService(
		repository.NewMockUnitOfWork(),
		storage.NewMockStorage(),
		defaultCfg,
	)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, "video.mp4", "video/mp4", 10000000, "abc", []string{})

	// Assert
	assert.Error(t, err)
	require.ErrorIs(t, err, domain.ErrFileSizeTooBig)
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}

func TestFileService_RequestUploadFile_NoExtension(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileService := file.NewFileService(
		repository.NewMockUnitOfWork(),
		storage.NewMockStorage(),
		defaultCfg,
	)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, "video", "video/mp4", 1024, "abc", []string{})

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidFileType))
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}

func TestFileService_RequestUploadFile_CreateRepoFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	fileService := file.NewFileService(mockUow, mockStorage, defaultCfg)

	createErr := errors.New("db error")

	mockUow.GetFileRepoMock().
		On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(createErr)

	mockUow.On("Execute", ctx, mock.Anything).Return(createErr)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, "video.mp4", "video/mp4", 1024, "abc", []string{})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}

func TestFileService_RequestUploadFile_FindByNamesFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	fileService := file.NewFileService(mockUow, mockStorage, defaultCfg)

	tags := []string{"football"}
	findErr := errors.New("db error finding tags")

	mockUow.GetFileRepoMock().
		On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(map[string]uuid.UUID{}, findErr)

	mockUow.On("Execute", ctx, mock.Anything).Return(findErr)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, "video.mp4", "video/mp4", 1024, "abc", tags)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}

func TestFileService_RequestUploadFile_CreateManyFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	fileService := file.NewFileService(mockUow, mockStorage, defaultCfg)

	tags := []string{"football"}
	tagID := uuid.New()
	tagMap := map[string]uuid.UUID{
		"football": tagID,
	}
	createManyErr := errors.New("db error creating file tags")

	mockUow.GetFileRepoMock().
		On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	mockUow.GetTagRepoMock().
		On("FindByNames", ctx, tags).
		Return(tagMap, nil)

	mockUow.GetFileTagRepoMock().
		On("CreateMany", ctx, mock.Anything, mock.Anything).
		Return(0, createManyErr)

	mockUow.On("Execute", ctx, mock.Anything).Return(createManyErr)

	// Act
	fileID, url, headers, expiresAt, err :=
		fileService.RequestUploadFile(ctx, "video.mp4", "video/mp4", 1024, "abc", tags)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, fileID)
	assert.Nil(t, url)
	assert.Nil(t, headers)
	assert.Nil(t, expiresAt)
}
