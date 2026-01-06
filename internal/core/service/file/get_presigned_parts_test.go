package file_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"score-play/internal/adapters/repository"
	"score-play/internal/adapters/storage"
	"score-play/internal/config"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/file"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var defaultCfg = config.FileUploadConfig{
	SingleUploadMaxSize:    10000,
	MultipartUploadMaxSize: 1000000,
	PartSize:               5000,
	SessionTTL:             time.Hour,
}

func TestFileService_GetPresignedParts_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	expectedErr := errors.New("session not found")

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(&domain.UploadSession{}, expectedErr)

	result, err := service.GetPresignedParts(ctx, sessionID, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
	mockUow.AssertExpectations(t)
}

func TestFileService_GetPresignedParts_FileMetadataNotFound(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()
	expectedErr := errors.New("file metadata not found")

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(&domain.UploadSession{FileID: fileID}, nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(&domain.FileMetadata{}, expectedErr)

	mockUow.GetUploadSessionRepoMock().
		On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(nil)

	result, err := service.GetPresignedParts(ctx, sessionID, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
	mockUow.AssertExpectations(t)
}

func TestFileService_GetPresignedParts_PresignedURLGenerationFails(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()
	storageErr := errors.New("storage error")

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "upload123",
	}

	fileMetadata := &domain.FileMetadata{
		ID:         fileID,
		StorageKey: "files/video.mp4",
		MimeType:   "video/mp4",
	}

	parts := []domain.UploadPart{
		{PartNumber: 1, ContentLength: 1000, ChecksumSHA256: "checksum"},
	}

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetUploadSessionRepoMock().
		On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(fileMetadata, nil)

	mockStorage.
		On("GeneratePresignedURLForPart",
			ctx,
			fileMetadata.StorageKey,
			1,
			session.ProviderUploadID,
			fileMetadata.MimeType,
			int64(1000),
			"checksum",
		).
		Return("", map[string]string{}, &time.Time{}, storageErr)

	result, err := service.GetPresignedParts(ctx, sessionID, parts)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, storageErr)

	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_GetPresignedParts_UpdateExpiresAtFails(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	updateErr := errors.New("update failed")
	fileID := uuid.New()

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "upload123",
	}

	parts := []domain.UploadPart{
		{PartNumber: 1, ContentLength: 1000, ChecksumSHA256: "checksum"},
	}

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetUploadSessionRepoMock().
		On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(updateErr)

	result, err := service.GetPresignedParts(ctx, sessionID, parts)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, updateErr)

	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_GetPresignedParts_EmptyPartsList(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "upload123",
	}

	fileMetadata := &domain.FileMetadata{
		ID:         fileID,
		StorageKey: "files/video.mp4",
		MimeType:   "video/mp4",
	}

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(fileMetadata, nil)

	mockUow.GetUploadSessionRepoMock().
		On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(nil)

	result, err := service.GetPresignedParts(ctx, sessionID, []domain.UploadPart{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)

	mockUow.AssertExpectations(t)
}

func TestFileService_GetPresignedParts_LargeNumberOfParts(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()
	numParts := 50

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "upload123",
	}

	fileMetadata := &domain.FileMetadata{
		ID:         fileID,
		StorageKey: "files/large.mp4",
		MimeType:   "video/mp4",
	}

	parts := make([]domain.UploadPart, numParts)
	for i := 0; i < numParts; i++ {
		parts[i] = domain.UploadPart{
			PartNumber:     i + 1,
			ContentLength:  int64((i + 1) * 1000),
			ChecksumSHA256: fmt.Sprintf("checksum%d", i+1),
		}
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(fileMetadata, nil)

	for i := 0; i < numParts; i++ {
		mockStorage.
			On("GeneratePresignedURLForPart",
				ctx,
				fileMetadata.StorageKey,
				i+1,
				session.ProviderUploadID,
				fileMetadata.MimeType,
				int64((i+1)*1000),
				fmt.Sprintf("checksum%d", i+1),
			).
			Return(fmt.Sprintf("https://storage.example.com/part%d", i+1), map[string]string{}, &expiresAt, nil)
	}

	mockUow.GetUploadSessionRepoMock().
		On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(nil)

	result, err := service.GetPresignedParts(ctx, sessionID, parts)

	assert.NoError(t, err)
	assert.Len(t, result, numParts)

	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
