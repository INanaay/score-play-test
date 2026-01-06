package file_test

import (
	"context"
	"errors"
	"score-play/internal/adapters/repository"
	"score-play/internal/adapters/storage"
	"score-play/internal/config"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/file"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFileService_GetFile_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()
	tagID1 := uuid.New()
	tagID2 := uuid.New()

	metadata := domain.FileMetadata{
		ID:         fileID,
		Filename:   "test-file.pdf",
		StorageKey: "storage-key",
		Status:     domain.FileStatusCompleted,
	}

	fileTags := []domain.FileTag{
		{TagID: tagID1, FileID: fileID},
		{TagID: tagID2, FileID: fileID},
	}

	tags := []domain.Tag{
		{ID: tagID1, Name: "tag1"},
		{ID: tagID2, Name: "tag2"},
	}

	downloadURL := "https://example.com/download"
	expiresAt := time.Now().Add(1 * time.Hour)

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()
	mockTagRepo := mockUow.GetTagRepoMock()

	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)
	mockFileTagRepo.On("FindByFileID", ctx, fileID).Return(fileTags, nil)
	mockTagRepo.On("FindByIDs", ctx, []uuid.UUID{tagID1, tagID2}).Return(tags, nil)
	mockStorage.On("GeneratePresignedURLForDownload", ctx, metadata.StorageKey).Return(downloadURL, &expiresAt, nil)

	// Act
	download, filename, resultTags, resultExpiresAt, err := service.GetFile(ctx, fileID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, download)
	assert.Equal(t, downloadURL, *download)
	assert.NotNil(t, filename)
	assert.Equal(t, metadata.Filename, *filename)
	assert.Equal(t, tags, resultTags)
	assert.NotNil(t, resultExpiresAt)
	assert.Equal(t, expiresAt, *resultExpiresAt)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_GetFile_FileNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()
	expectedError := errors.New("file not found")

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileRepo.On("FindById", ctx, fileID).Return(&domain.FileMetadata{}, expectedError)

	// Act
	download, filename, tags, expiresAt, err := service.GetFile(ctx, fileID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, download)
	assert.Nil(t, filename)
	assert.Nil(t, tags)
	assert.Nil(t, expiresAt)
	mockFileRepo.AssertExpectations(t)
}

func TestFileService_GetFile_FileUploading(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()

	metadata := domain.FileMetadata{
		ID:         fileID,
		Filename:   "test-file.pdf",
		StorageKey: "storage-key",
		Status:     domain.FileStatusUploading,
	}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)

	// Act
	download, filename, tags, expiresAt, err := service.GetFile(ctx, fileID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrFileNotReady, err)
	assert.Nil(t, download)
	assert.Nil(t, filename)
	assert.Nil(t, tags)
	assert.Nil(t, expiresAt)
	mockFileRepo.AssertExpectations(t)
}

func TestFileService_GetFile_FileFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()

	metadata := domain.FileMetadata{
		ID:         fileID,
		Filename:   "test-file.pdf",
		StorageKey: "storage-key",
		Status:     domain.FileStatusFailed,
	}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)

	// Act
	download, filename, tags, expiresAt, err := service.GetFile(ctx, fileID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrFileUploadFailed, err)
	assert.Nil(t, download)
	assert.Nil(t, filename)
	assert.Nil(t, tags)
	assert.Nil(t, expiresAt)
	mockFileRepo.AssertExpectations(t)
}

func TestFileService_GetFile_FindFileTagsError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()
	expectedError := errors.New("database error")

	metadata := domain.FileMetadata{
		ID:         fileID,
		Filename:   "test-file.pdf",
		StorageKey: "storage-key",
		Status:     domain.FileStatusCompleted,
	}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)
	mockFileTagRepo.On("FindByFileID", ctx, fileID).Return([]domain.FileTag{}, expectedError)

	// Act
	download, filename, tags, expiresAt, err := service.GetFile(ctx, fileID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, download)
	assert.Nil(t, filename)
	assert.Nil(t, tags)
	assert.Nil(t, expiresAt)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
}

func TestFileService_GetFile_FindTagsByIDsError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()
	tagID := uuid.New()
	expectedError := errors.New("database error")

	metadata := domain.FileMetadata{
		ID:         fileID,
		Filename:   "test-file.pdf",
		StorageKey: "storage-key",
		Status:     domain.FileStatusCompleted,
	}

	fileTags := []domain.FileTag{
		{TagID: tagID, FileID: fileID},
	}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()
	mockTagRepo := mockUow.GetTagRepoMock()

	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)
	mockFileTagRepo.On("FindByFileID", ctx, fileID).Return(fileTags, nil)
	mockTagRepo.On("FindByIDs", ctx, []uuid.UUID{tagID}).Return([]domain.Tag{}, expectedError)

	// Act
	download, filename, tags, expiresAt, err := service.GetFile(ctx, fileID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, download)
	assert.Nil(t, filename)
	assert.Nil(t, tags)
	assert.Nil(t, expiresAt)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockTagRepo.AssertExpectations(t)
}

func TestFileService_GetFile_GeneratePresignedURLError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()
	expectedError := errors.New("storage error")

	metadata := domain.FileMetadata{
		ID:         fileID,
		Filename:   "test-file.pdf",
		StorageKey: "storage-key",
		Status:     domain.FileStatusCompleted,
	}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()
	mockTagRepo := mockUow.GetTagRepoMock()

	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)
	mockFileTagRepo.On("FindByFileID", ctx, fileID).Return([]domain.FileTag{}, nil)
	mockTagRepo.On("FindByIDs", ctx, mock.Anything).Return([]domain.Tag{}, nil)
	mockStorage.On("GeneratePresignedURLForDownload", ctx, metadata.StorageKey).Return("", &time.Time{}, expectedError)

	// Act
	download, filename, tags, expiresAt, err := service.GetFile(ctx, fileID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, download)
	assert.Nil(t, filename)
	assert.Nil(t, tags)
	assert.Nil(t, expiresAt)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_GetFile_EmptyDownloadURL(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	cfg := config.FileUploadConfig{}
	service := file.NewFileService(mockUow, mockStorage, cfg)

	fileID := uuid.New()

	metadata := domain.FileMetadata{
		ID:         fileID,
		Filename:   "test-file.pdf",
		StorageKey: "storage-key",
		Status:     domain.FileStatusCompleted,
	}

	expiresAt := time.Now().Add(1 * time.Hour)

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()
	mockTagRepo := mockUow.GetTagRepoMock()

	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)
	mockFileTagRepo.On("FindByFileID", ctx, fileID).Return([]domain.FileTag{}, nil)
	mockTagRepo.On("FindByIDs", ctx, mock.Anything).Return([]domain.Tag{}, nil)
	mockStorage.On("GeneratePresignedURLForDownload", ctx, metadata.StorageKey).Return("", &expiresAt, nil)

	// Act
	download, filename, tags, resTime, err := service.GetFile(ctx, fileID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, download)
	assert.Nil(t, filename)
	assert.Nil(t, tags)
	assert.Nil(t, resTime)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
