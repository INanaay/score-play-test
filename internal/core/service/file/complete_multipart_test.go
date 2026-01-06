package file_test

import (
	"context"
	"score-play/internal/adapters/repository"
	"score-play/internal/adapters/storage"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/file"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFileService_CompleteMultipartUpload_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	fileID := uuid.New()
	uploadID := "upload-id-123"
	storageKey := "path/to/file.mp4"

	session := &domain.UploadSession{ID: sessionID, FileID: fileID, ProviderUploadID: uploadID}
	fileMetadata := &domain.FileMetadata{ID: fileID, StorageKey: storageKey}
	parts := []domain.UploadPart{
		{PartNumber: 1, ETag: "etag1"},
		{PartNumber: 2, ETag: "etag2"},
	}

	mockUow.GetUploadSessionRepoMock().On("FindByIDAndActive", ctx, sessionID).Return(session, nil)
	mockUow.GetUploadSessionRepoMock().On("UpdateExpiresAt", ctx, sessionID, mock.Anything).Return(nil)
	mockUow.GetFileRepoMock().On("FindById", ctx, fileID).Return(fileMetadata, nil)
	mockStorage.On("ListPartsPaginated", ctx, storageKey, uploadID, 1000, 0).Return(parts, 0, nil)
	mockStorage.On("CompleteMultipartUpload", ctx, storageKey, uploadID, parts).Return(nil)

	// Act
	id, err := service.CompleteMultipartUpload(ctx, sessionID, parts)

	// Assert
	assert.NoError(t, err)
	require.Equal(t, fileID, *id)
	mockUow.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_CompleteMultipartUpload_DuplicateParts(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	parts := []domain.UploadPart{
		{PartNumber: 1, ETag: "etag1"},
		{PartNumber: 1, ETag: "etag2"},
	}

	mockUow.GetUploadSessionRepoMock().On("FindByIDAndActive", ctx, sessionID).Return(&domain.UploadSession{}, nil)
	mockUow.GetUploadSessionRepoMock().On("UpdateExpiresAt", ctx, sessionID, mock.Anything).Return(nil)
	mockUow.GetFileRepoMock().On("FindById", ctx, mock.Anything).Return(&domain.FileMetadata{}, nil)

	// Act
	id, err := service.CompleteMultipartUpload(ctx, sessionID, parts)

	// Assert
	assert.ErrorIs(t, err, domain.ErrDuplicatePart)
	require.Nil(t, id)
}

func TestFileService_CompleteMultipartUpload_ETagMismatch(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	storageKey := "file.mp4"
	uploadID := "up-id"
	parts := []domain.UploadPart{{PartNumber: 1, ETag: "expected-etag"}}

	mockUow.GetUploadSessionRepoMock().On("FindByIDAndActive", ctx, sessionID).
		Return(&domain.UploadSession{ProviderUploadID: uploadID}, nil)
	mockUow.GetUploadSessionRepoMock().On("UpdateExpiresAt", ctx, sessionID, mock.Anything).
		Return(nil)
	mockUow.GetFileRepoMock().On("FindById", ctx, mock.Anything).
		Return(&domain.FileMetadata{StorageKey: storageKey}, nil)
	mockStorage.On("ListPartsPaginated", ctx, storageKey, uploadID, 1000, 0).
		Return([]domain.UploadPart{{PartNumber: 1, ETag: "wrong-etag"}}, 0, nil)

	// Act
	id, err := service.CompleteMultipartUpload(ctx, sessionID, parts)

	// Assert
	assert.ErrorIs(t, err, domain.ErrMismatchETag)
	require.Nil(t, id)
}

func TestFileService_CompleteMultipartUpload_CountMismatch(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockUow := repository.NewMockUnitOfWork()
	mockStorage := storage.NewMockStorage()
	service := file.NewFileService(mockUow, mockStorage, defaultCfg)

	sessionID := uuid.New()
	parts := []domain.UploadPart{
		{PartNumber: 1, ETag: "etag1"},
		{PartNumber: 2, ETag: "etag2"},
	}

	mockUow.GetUploadSessionRepoMock().On("FindByIDAndActive", ctx, sessionID).Return(&domain.UploadSession{}, nil)
	mockUow.GetUploadSessionRepoMock().On("UpdateExpiresAt", ctx, sessionID, mock.Anything).Return(nil)
	mockUow.GetFileRepoMock().On("FindById", ctx, mock.Anything).Return(&domain.FileMetadata{}, nil)
	mockStorage.On("ListPartsPaginated", ctx, mock.Anything, mock.Anything, 1000, 0).
		Return([]domain.UploadPart{{PartNumber: 1, ETag: "etag1"}}, 0, nil)

	// Act
	id, err := service.CompleteMultipartUpload(ctx, sessionID, parts)

	// Assert
	assert.ErrorIs(t, err, domain.ErrMismatchNBParts)
	require.Nil(t, id)
}
