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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCleanupService_FinalizeUpload_MultipartSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, config.FileUploadConfig{})

	metadata := domain.FileMetadata{ID: uuid.New()}
	session := &domain.UploadSession{ID: uuid.New()}
	eventType := domain.EventTypeMultipartUploadComplete

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileRepo := mockUow.GetFileRepoMock()

	mockUploadSessionRepo.On("FindByFileID", ctx, metadata.ID).Return(session, nil)
	mockUow.On("Execute", ctx, mock.Anything).Return(nil)
	mockUploadSessionRepo.On("UpdateStatusByFileID", ctx, metadata.ID, domain.UploadSessionStatusCompleted).Return(nil)
	mockFileRepo.On("UpdateStatus", ctx, metadata.ID, domain.FileStatusCompleted).Return(nil)

	// Act
	err := service.FinalizeUpload(ctx, metadata, nil, eventType)

	// Assert
	assert.NoError(t, err)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestCleanupService_FinalizeUpload_MultipartFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, config.FileUploadConfig{})

	metadata := domain.FileMetadata{ID: uuid.New(), StorageKey: "storage-key"}
	session := &domain.UploadSession{ID: uuid.New(), ProviderUploadID: "provider-id"}
	eventType := domain.EventTypeMultipartUploadComplete
	uploadErr := errors.New("s3 failure")

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockUploadSessionRepo.On("FindByFileID", ctx, metadata.ID).Return(session, nil)
	mockUow.On("Execute", ctx, mock.Anything).Return(nil)
	mockUploadSessionRepo.On("UpdateStatusByFileID", ctx, metadata.ID, domain.UploadSessionStatusAborted).Return(nil)
	mockFileRepo.On("UpdateStatus", ctx, metadata.ID, domain.FileStatusFailed).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, metadata.ID).Return(nil)
	mockFileRepo.On("Delete", ctx, metadata.ID).Return(nil)
	mockStorage.On("AbortMultipartUpload", ctx, metadata.StorageKey, "provider-id").Return(nil)

	// Act
	err := service.FinalizeUpload(ctx, metadata, uploadErr, eventType)

	// Assert
	assert.NoError(t, err)
	mockUploadSessionRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestCleanupService_FinalizeUpload_SimpleFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, config.FileUploadConfig{})

	metadata := domain.FileMetadata{ID: uuid.New(), StorageKey: "key"}
	uploadErr := errors.New("network error")

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockUow.On("Execute", ctx, mock.Anything).Return(nil)
	mockFileRepo.On("UpdateStatus", ctx, metadata.ID, domain.FileStatusFailed).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, metadata.ID).Return(nil)
	mockFileRepo.On("Delete", ctx, metadata.ID).Return(nil)
	mockStorage.On("DeleteObject", ctx, metadata.StorageKey).Return(nil)

	// Act
	err := service.FinalizeUpload(ctx, metadata, uploadErr, domain.EventTypeSimpleUploadComplete)

	// Assert
	assert.NoError(t, err)
	mockFileRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
