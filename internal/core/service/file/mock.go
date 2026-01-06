package file

import (
	"context"
	"score-play/internal/core/domain"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockFileService is a mock implementation of FileService
type MockFileService struct {
	mock.Mock
}

// NewMockFileService creates a new MockFileService
func NewMockFileService() *MockFileService {
	return &MockFileService{}
}

func (m *MockFileService) RequestUploadFile(ctx context.Context, fileName string, contentType string, sizeBytes int64, checksumSha256 string, tags []string) (*uuid.UUID, *string, map[string]string, *time.Time, error) {
	args := m.Called(ctx, fileName, contentType, sizeBytes, checksumSha256, tags)
	return args.Get(0).(*uuid.UUID),
		args.Get(1).(*string),
		args.Get(2).(map[string]string),
		args.Get(3).(*time.Time),
		args.Error(4)
}

func (m *MockFileService) RequestUploadMultipartFile(ctx context.Context, fileName string, contentType string, sizeBytes int64, checksumSha256 string, tags []string) (*uuid.UUID, int, error) {
	args := m.Called(ctx, fileName, contentType, sizeBytes, checksumSha256, tags)
	return args.Get(0).(*uuid.UUID), args.Int(1), args.Error(2)
}

func (m *MockFileService) GetPresignedParts(ctx context.Context, sessionID uuid.UUID, parts []domain.UploadPart) ([]domain.UploadPart, error) {
	args := m.Called(ctx, sessionID, parts)
	return args.Get(0).([]domain.UploadPart), args.Error(1)
}

func (m *MockFileService) ListParts(ctx context.Context, sessionID uuid.UUID, maxParts int, partNumberMarker int) ([]domain.UploadPart, int, error) {
	args := m.Called(ctx, sessionID, maxParts, partNumberMarker)
	return args.Get(0).([]domain.UploadPart), args.Int(1), args.Error(2)
}

func (m *MockFileService) CompleteMultipartUpload(ctx context.Context, sessionID uuid.UUID, parts []domain.UploadPart) (*uuid.UUID, error) {
	args := m.Called(ctx, sessionID, parts)
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

func (m *MockFileService) GetFile(ctx context.Context, fileID uuid.UUID) (*string, *string, []domain.Tag, *time.Time, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).(*string), args.Get(1).(*string), args.Get(2).([]domain.Tag), args.Get(3).(*time.Time), args.Error(4)
}

func (m *MockFileService) FinalizeUpload(ctx context.Context, metadata domain.FileMetadata, err error, eventType domain.EventType) error {
	args := m.Called(ctx, metadata, err, eventType)
	return args.Error(0)
}
