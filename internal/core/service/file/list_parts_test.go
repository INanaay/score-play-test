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

func TestFileService_ListParts_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()
	maxParts := 10
	partNumberMarker := 0

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider_upload_123",
		Status:           domain.UploadSessionStatusOpen,
	}

	fileMetadata := &domain.FileMetadata{
		ID:         fileID,
		StorageKey: "files/video123.mp4",
		MimeType:   "video/mp4",
	}

	expectedParts := []domain.UploadPart{
		{PartNumber: 1, ETag: "etag1"},
		{PartNumber: 2, ETag: "etag2"},
		{PartNumber: 3, ETag: "etag3"},
	}
	expectedNewMarker := 3

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(fileMetadata, nil)

	mockStorage.
		On(
			"ListPartsPaginated",
			ctx,
			fileMetadata.StorageKey,
			session.ProviderUploadID,
			maxParts,
			partNumberMarker,
		).
		Return(expectedParts, expectedNewMarker, nil)

	mockUow.GetUploadSessionRepoMock().
		On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(nil)

	// Act
	result, newMarker, err := service.ListParts(
		ctx,
		sessionID,
		maxParts,
		partNumberMarker,
	)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)
	assert.Equal(t, expectedNewMarker, newMarker)
	assert.Equal(t, expectedParts, result)

	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_ListParts_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()

	expectedErr := errors.New("session not found")

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(&domain.UploadSession{}, expectedErr)

	result, marker, err := service.ListParts(ctx, sessionID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, marker)
	assert.Equal(t, expectedErr, err)
	mockUow.AssertExpectations(t)
}

func TestFileService_ListParts_FileMetadataNotFound(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider_upload_123",
		Status:           domain.UploadSessionStatusOpen,
	}

	expectedErr := errors.New("file metadata not found")

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(&domain.FileMetadata{}, expectedErr)

	result, marker, err := service.ListParts(ctx, sessionID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, marker)
	assert.Equal(t, expectedErr, err)
	mockUow.AssertExpectations(t)
}

func TestFileService_ListParts_StorageListPartsFails(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider_upload_123",
		Status:           domain.UploadSessionStatusOpen,
	}

	fileMetadata := &domain.FileMetadata{
		ID:         fileID,
		StorageKey: "files/video123.mp4",
	}

	storageErr := errors.New("storage error")

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(fileMetadata, nil)

	mockStorage.
		On("ListPartsPaginated",
			ctx,
			fileMetadata.StorageKey,
			session.ProviderUploadID,
			10,
			0,
		).
		Return([]domain.UploadPart{}, 0, storageErr)

	result, marker, err := service.ListParts(ctx, sessionID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, marker)
	assert.Equal(t, storageErr, err)
	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_ListParts_UpdateExpiresAtFails(t *testing.T) {
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()

	session := &domain.UploadSession{
		ID:               sessionID,
		FileID:           fileID,
		ProviderUploadID: "provider_upload_123",
		Status:           domain.UploadSessionStatusOpen,
	}

	fileMetadata := &domain.FileMetadata{
		ID:         fileID,
		StorageKey: "files/video123.mp4",
	}

	parts := []domain.UploadPart{{PartNumber: 1, ETag: "etag1"}}
	updateErr := errors.New("update failed")

	mockUow.GetUploadSessionRepoMock().
		On("FindByIDAndActive", ctx, sessionID).
		Return(session, nil)

	mockUow.GetFileRepoMock().
		On("FindById", ctx, fileID).
		Return(fileMetadata, nil)

	mockStorage.
		On("ListPartsPaginated",
			ctx,
			fileMetadata.StorageKey,
			session.ProviderUploadID,
			10,
			0,
		).
		Return(parts, 1, nil)

	mockUow.GetUploadSessionRepoMock().
		On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(updateErr)

	result, marker, err := service.ListParts(ctx, sessionID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, marker)
	assert.Equal(t, updateErr, err)
	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
