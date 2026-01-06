package cleanup_test

import (
	"context"
	"errors"
	"log/slog"
	"score-play/internal/adapters/repository"
	"score-play/internal/adapters/storage"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/cleanup"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCleanupService_CleanupExpiredFiles_NoExpiredFiles(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	mockFileRepo := mockUow.GetFileRepoMock()

	mockFileRepo.On("FindExpired", ctx, now).Return([]domain.FileMetadata{}, nil)

	// Act
	err := service.CleanupExpiredFiles(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockFileRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredFiles_SuccessWithSession(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID := uuid.New()
	sessionID := uuid.New()

	file := domain.FileMetadata{
		ID:         fileID,
		StorageKey: "storage-key",
	}

	session := domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider-upload-id",
	}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockFileRepo.On("FindExpired", ctx, now).Return([]domain.FileMetadata{file}, nil)
	mockUploadSessionRepo.On("FindByFileID", ctx, fileID).Return(&session, nil)

	mockFileRepo.On("UpdateStatus", ctx, session.FileID, domain.FileStatusFailed).Return(nil)
	mockFileRepo.On("Delete", ctx, session.FileID).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, session.FileID).Return(nil)
	mockUploadSessionRepo.On("UpdateStatus", ctx, session.ID, domain.UploadSessionStatusAborted).Return(nil)
	mockStorage.On("AbortMultipartUpload", ctx, file.StorageKey, session.ProviderUploadID).Return(nil)

	mockUow.On("Execute", ctx, mock.Anything).Return(nil)

	// Act
	err := service.CleanupExpiredFiles(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockFileRepo.AssertExpectations(t)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredFiles_SuccessWithoutSession(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID := uuid.New()
	file := domain.FileMetadata{ID: fileID, StorageKey: "simple-key"}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockFileRepo.On("FindExpired", ctx, now).Return([]domain.FileMetadata{file}, nil)
	mockUploadSessionRepo.On("FindByFileID", ctx, fileID).Return((*domain.UploadSession)(nil), domain.ErrSessionNotFound)

	mockUow.On("Execute", ctx, mock.Anything).Return(nil)

	mockFileRepo.On("UpdateStatus", ctx, fileID, domain.FileStatusFailed).Return(nil)
	mockFileRepo.On("Delete", ctx, fileID).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, fileID).Return(nil)
	mockStorage.On("DeleteObject", ctx, file.StorageKey).Return(nil)

	err := service.CleanupExpiredFiles(ctx, now)

	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
	mockUploadSessionRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredFiles_FindExpiredError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	expectedError := errors.New("database error")

	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileRepo.On("FindExpired", ctx, now).Return([]domain.FileMetadata{}, expectedError)

	// Act
	err := service.CleanupExpiredFiles(ctx, now)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFileRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredFiles_FindByFileIDError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID := uuid.New()
	file := domain.FileMetadata{ID: fileID}
	expectedError := errors.New("unexpected error")

	mockFileRepo := mockUow.GetFileRepoMock()
	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()

	mockFileRepo.On("FindExpired", ctx, now).Return([]domain.FileMetadata{file}, nil)
	mockUploadSessionRepo.On("FindByFileID", ctx, fileID).Return((*domain.UploadSession)(nil), expectedError)

	// Act
	err := service.CleanupExpiredFiles(ctx, now)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFileRepo.AssertExpectations(t)
	mockUploadSessionRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredFiles_PartialFailure(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID1 := uuid.New()
	fileID2 := uuid.New()
	sessionID2 := uuid.New()

	file1 := domain.FileMetadata{ID: fileID1}
	file2 := domain.FileMetadata{ID: fileID2, StorageKey: "storage-key-2"}

	session1 := domain.UploadSession{ID: uuid.New(), FileID: fileID1}
	session2 := domain.UploadSession{
		ID:               sessionID2,
		FileID:           fileID2,
		ProviderUploadID: "provider-upload-id-2",
	}

	mockFileRepo := mockUow.GetFileRepoMock()
	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockFileRepo.On("FindExpired", ctx, now).Return([]domain.FileMetadata{file1, file2}, nil)

	// First file fails during transaction
	mockUploadSessionRepo.On("FindByFileID", ctx, fileID1).Return(&session1, nil)
	mockFileRepo.On("UpdateStatus", ctx, session1.FileID, domain.FileStatusFailed).Return(errors.New("update error")).Once()
	mockUow.On("Execute", ctx, mock.Anything).Return(errors.New("transaction error")).Once()

	// Second file succeeds
	mockUploadSessionRepo.On("FindByFileID", ctx, fileID2).Return(&session2, nil)
	mockFileRepo.On("UpdateStatus", ctx, session2.FileID, domain.FileStatusFailed).Return(nil).Once()
	mockFileRepo.On("Delete", ctx, session2.FileID).Return(nil).Once()
	mockFileTagRepo.On("DeleteByFileID", ctx, session2.FileID).Return(nil).Once()
	mockUploadSessionRepo.On("UpdateStatus", ctx, session2.ID, domain.UploadSessionStatusAborted).Return(nil).Once()
	mockStorage.On("AbortMultipartUpload", ctx, file2.StorageKey, session2.ProviderUploadID).Return(nil).Once()
	mockUow.On("Execute", ctx, mock.Anything).Return(nil).Once()

	// Act
	err := service.CleanupExpiredFiles(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockFileRepo.AssertExpectations(t)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
