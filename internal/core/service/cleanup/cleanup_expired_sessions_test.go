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

func TestCleanupService_CleanupExpiredSessions_NoExpiredSessions(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()

	mockUploadSessionRepo.On("FindAllExpired", ctx, now).Return([]domain.UploadSession{}, nil)

	// Act
	err := service.CleanupExpiredSessions(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockUploadSessionRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredSessions_SingleSession(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID := uuid.New()
	sessionID := uuid.New()

	session := domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider-upload-id",
	}

	metadata := domain.FileMetadata{
		ID:         fileID,
		StorageKey: "storage-key",
	}

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockUploadSessionRepo.On("FindAllExpired", ctx, now).Return([]domain.UploadSession{session}, nil)
	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)
	mockFileRepo.On("UpdateStatus", ctx, fileID, domain.FileStatusFailed).Return(nil)
	mockFileRepo.On("Delete", ctx, fileID).Return(nil)
	mockUploadSessionRepo.On("UpdateStatus", ctx, sessionID, domain.UploadSessionStatusAborted).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, fileID).Return(nil)
	mockStorage.On("AbortMultipartUpload", ctx, metadata.StorageKey, session.ProviderUploadID).Return(nil)
	mockUow.On("Execute", ctx, mock.Anything).Return(nil)

	// Act
	err := service.CleanupExpiredSessions(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredSessions_MultipleSessions(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID1 := uuid.New()
	fileID2 := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()

	session1 := domain.UploadSession{
		ID:               sessionID1,
		FileID:           fileID1,
		ProviderUploadID: "provider-upload-id-1",
	}

	session2 := domain.UploadSession{
		ID:               sessionID2,
		FileID:           fileID2,
		ProviderUploadID: "provider-upload-id-2",
	}

	metadata1 := domain.FileMetadata{
		ID:         fileID1,
		StorageKey: "storage-key-1",
	}

	metadata2 := domain.FileMetadata{
		ID:         fileID2,
		StorageKey: "storage-key-2",
	}

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockUploadSessionRepo.On("FindAllExpired", ctx, now).Return([]domain.UploadSession{session1, session2}, nil)

	// Session 1
	mockFileRepo.On("FindById", ctx, fileID1).Return(&metadata1, nil)
	mockFileRepo.On("UpdateStatus", ctx, fileID1, domain.FileStatusFailed).Return(nil)
	mockFileRepo.On("Delete", ctx, fileID1).Return(nil)
	mockUploadSessionRepo.On("UpdateStatus", ctx, sessionID1, domain.UploadSessionStatusAborted).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, fileID1).Return(nil)
	mockStorage.On("AbortMultipartUpload", ctx, metadata1.StorageKey, session1.ProviderUploadID).Return(nil)

	// Session 2
	mockFileRepo.On("FindById", ctx, fileID2).Return(&metadata2, nil)
	mockFileRepo.On("UpdateStatus", ctx, fileID2, domain.FileStatusFailed).Return(nil)
	mockFileRepo.On("Delete", ctx, fileID2).Return(nil)
	mockUploadSessionRepo.On("UpdateStatus", ctx, sessionID2, domain.UploadSessionStatusAborted).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, fileID2).Return(nil)
	mockStorage.On("AbortMultipartUpload", ctx, metadata2.StorageKey, session2.ProviderUploadID).Return(nil)

	mockUow.On("Execute", ctx, mock.Anything).Return(nil).Times(2)

	// Act
	err := service.CleanupExpiredSessions(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredSessions_FindAllExpiredError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	expectedError := errors.New("database error")

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockUploadSessionRepo.On("FindAllExpired", ctx, now).Return([]domain.UploadSession{}, expectedError)

	// Act
	err := service.CleanupExpiredSessions(ctx, now)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockUploadSessionRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredSessions_FindByIdError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID := uuid.New()
	sessionID := uuid.New()
	expectedError := errors.New("file not found")

	session := domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider-upload-id",
	}

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileRepo := mockUow.GetFileRepoMock()

	mockUploadSessionRepo.On("FindAllExpired", ctx, now).Return([]domain.UploadSession{session}, nil)
	mockFileRepo.On("FindById", ctx, fileID).Return(&domain.FileMetadata{}, expectedError)

	// Act
	err := service.CleanupExpiredSessions(ctx, now)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredSessions_ExecuteError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID := uuid.New()
	sessionID := uuid.New()
	expectedError := errors.New("transaction failed")

	session := domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider-upload-id",
	}

	metadata := domain.FileMetadata{
		ID:         fileID,
		StorageKey: "storage-key",
	}

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockUploadSessionRepo.On("FindAllExpired", ctx, now).Return([]domain.UploadSession{session}, nil)
	mockFileRepo.On("FindById", ctx, fileID).Return(&metadata, nil)

	mockFileRepo.On("UpdateStatus", ctx, fileID, domain.FileStatusFailed).Return(nil)
	mockFileRepo.On("Delete", ctx, fileID).Return(expectedError) // Ici on fait échouer
	mockUploadSessionRepo.On("UpdateStatus", ctx, sessionID, domain.UploadSessionStatusAborted).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, fileID).Return(nil)
	mockStorage.On("AbortMultipartUpload", ctx, metadata.StorageKey, session.ProviderUploadID).Return(nil)

	mockUow.On("Execute", ctx, mock.Anything).Return(expectedError)

	// Act
	err := service.CleanupExpiredSessions(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupExpiredSessions_PartialFailure(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	logger := slog.Default()
	service := cleanup.NewCleanupService(mockUow, mockStorage, logger)

	now := time.Now()
	fileID1 := uuid.New()
	fileID2 := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()

	session1 := domain.UploadSession{
		ID:               sessionID1,
		FileID:           fileID1,
		ProviderUploadID: "provider-upload-id-1",
	}

	session2 := domain.UploadSession{
		ID:               sessionID2,
		FileID:           fileID2,
		ProviderUploadID: "provider-upload-id-2",
	}

	metadata1 := domain.FileMetadata{
		ID:         fileID1,
		StorageKey: "storage-key-1",
	}

	metadata2 := domain.FileMetadata{
		ID:         fileID2,
		StorageKey: "storage-key-2",
	}

	mockUploadSessionRepo := mockUow.GetUploadSessionRepoMock()
	mockFileRepo := mockUow.GetFileRepoMock()
	mockFileTagRepo := mockUow.GetFileTagRepoMock()

	mockUploadSessionRepo.On("FindAllExpired", ctx, now).Return([]domain.UploadSession{session1, session2}, nil)

	mockFileRepo.On("FindById", ctx, fileID1).Return(&metadata1, nil)
	mockFileRepo.On("UpdateStatus", ctx, fileID1, domain.FileStatusFailed).Return(errors.New("update failed")).Once()
	mockUow.On("Execute", ctx, mock.Anything).Return(errors.New("transaction failed")).Once()

	mockFileRepo.On("FindById", ctx, fileID2).Return(&metadata2, nil)
	mockFileRepo.On("UpdateStatus", ctx, fileID2, domain.FileStatusFailed).Return(nil)
	mockFileRepo.On("Delete", ctx, fileID2).Return(nil)
	mockUploadSessionRepo.On("UpdateStatus", ctx, sessionID2, domain.UploadSessionStatusAborted).Return(nil)
	mockFileTagRepo.On("DeleteByFileID", ctx, fileID2).Return(nil)
	mockStorage.On("AbortMultipartUpload", ctx, metadata2.StorageKey, session2.ProviderUploadID).Return(nil)
	mockUow.On("Execute", ctx, mock.Anything).Return(nil).Once()

	// Act
	err := service.CleanupExpiredSessions(ctx, now)

	// Assert
	assert.NoError(t, err)
	mockUploadSessionRepo.AssertExpectations(t)
	mockFileRepo.AssertExpectations(t)
	mockFileTagRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
