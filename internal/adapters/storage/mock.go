package storage

import (
	"context"
	"io"
	"score-play/internal/core/domain"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"
)

type MockStorage struct {
	mock.Mock
}

func NewMockStorage() *MockStorage {
	return &MockStorage{}
}

func (m *MockStorage) GeneratePresignedURLSimpleUpload(ctx context.Context, fileKey string, checksumSha256 string) (string, map[string]string, *time.Time, error) {
	args := m.Called(ctx, fileKey, checksumSha256)
	return args.String(0), args.Get(1).(map[string]string), args.Get(2).(*time.Time), args.Error(3)
}

func (m *MockStorage) InitMultipartUpload(ctx context.Context, fileName string, checksum string) (string, error) {
	args := m.Called(ctx, fileName, checksum)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetHeaderBytes(ctx context.Context, fileKey string, n int64) ([]byte, error) {
	args := m.Called(ctx, fileKey, n)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStorage) GeneratePresignedURLForPart(ctx context.Context, fileKey string, partNumber int, uploadID, mimeType string, contentLength int64, checksumSha256 string) (string, map[string]string, *time.Time, error) {
	args := m.Called(ctx, fileKey, partNumber, uploadID, mimeType, contentLength, checksumSha256)
	return args.String(0), args.Get(1).(map[string]string), args.Get(2).(*time.Time), args.Error(3)
}

func (m *MockStorage) CompleteMultipartUpload(ctx context.Context, fileKey string, uploadID string, parts []domain.UploadPart) error {
	args := m.Called(ctx, fileKey, uploadID, parts)
	return args.Error(0)
}

func (m *MockStorage) GetObject(ctx context.Context, fileKey string) (io.ReadCloser, error) {
	args := m.Called(ctx, fileKey)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) GetObjectInfo(ctx context.Context, fileKey string) (*minio.ObjectInfo, error) {
	args := m.Called(ctx, fileKey)
	return args.Get(0).(*minio.ObjectInfo), args.Error(1)
}

func (m *MockStorage) ListPartsPaginated(ctx context.Context, fileKey string, uploadID string, maxParts int, partNumberMarker int) ([]domain.UploadPart, int, error) {
	args := m.Called(ctx, fileKey, uploadID, maxParts, partNumberMarker)
	return args.Get(0).([]domain.UploadPart), args.Int(1), args.Error(2)
}

func (m *MockStorage) AbortMultipartUpload(ctx context.Context, fileKey string, uploadID string) error {
	args := m.Called(ctx, fileKey, uploadID)
	return args.Error(0)
}

func (m *MockStorage) DeleteObject(ctx context.Context, fileKey string) error {
	args := m.Called(ctx, fileKey)
	return args.Error(0)
}

func (m *MockStorage) GeneratePresignedURLForDownload(ctx context.Context, fileKey string) (string, *time.Time, error) {
	args := m.Called(ctx, fileKey)
	return args.Get(0).(string), args.Get(1).(*time.Time), args.Error(2)
}
