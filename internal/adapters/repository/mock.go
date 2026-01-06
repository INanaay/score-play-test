package repository

import (
	"context"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockTagRepository struct {
	mock.Mock
}

func NewMockTagRepository() *MockTagRepository {
	return &MockTagRepository{}
}

func (m *MockTagRepository) Create(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockTagRepository) CreateMany(ctx context.Context, tags []string) (int, error) {
	args := m.Called(ctx, tags)
	return args.Int(0), args.Error(1)
}

func (m *MockTagRepository) FindByName(ctx context.Context, name string) (*domain.Tag, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*domain.Tag), args.Error(1)
}

func (m *MockTagRepository) FindByNames(ctx context.Context, names []string) (map[string]uuid.UUID, error) {
	args := m.Called(ctx, names)
	return args.Get(0).(map[string]uuid.UUID), args.Error(1)
}

func (m *MockTagRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Tag, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.Tag), args.Error(1)
}

func (m *MockTagRepository) List(ctx context.Context, limit int, marker *string) ([]domain.Tag, *string, error) {
	args := m.Called(ctx, limit, marker)
	return args.Get(0).([]domain.Tag), args.Get(1).(*string), args.Error(2)
}

type MockFileRepository struct {
	mock.Mock
}

func NewMockFileRepository() *MockFileRepository {
	return &MockFileRepository{}
}

func (m *MockFileRepository) Create(ctx context.Context, id uuid.UUID, fileName, mimeType string, mediaType domain.FileType, size int64, status domain.FileStatus, checksum string, storageKey string) error {
	args := m.Called(ctx, id, fileName, mimeType, mediaType, size, status, checksum, storageKey)
	return args.Error(0)
}

func (m *MockFileRepository) FindById(ctx context.Context, id uuid.UUID) (*domain.FileMetadata, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.FileMetadata), args.Error(1)
}

func (m *MockFileRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) FindExpired(ctx context.Context, expirationTime time.Time) ([]domain.FileMetadata, error) {
	args := m.Called(ctx, expirationTime)
	return args.Get(0).([]domain.FileMetadata), args.Error(1)
}

type MockUploadSessionRepository struct {
	mock.Mock
}

func NewMockUploadSessionRepository() *MockUploadSessionRepository {
	return &MockUploadSessionRepository{}
}

func (m *MockUploadSessionRepository) Create(ctx context.Context, session domain.UploadSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) UpdateExpiresAt(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	args := m.Called(ctx, id, expiresAt)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) OpenSessionExists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockUploadSessionRepository) CloseSession(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) FindByIDAndActive(ctx context.Context, id uuid.UUID) (*domain.UploadSession, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.UploadSession, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) UpdateStatusByFileID(ctx context.Context, fileID uuid.UUID, status domain.UploadSessionStatus) error {
	args := m.Called(ctx, fileID, status)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) UpdateAllInactive(ctx context.Context, now time.Time) error {
	args := m.Called(ctx, now)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) (*domain.UploadSession, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).(*domain.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) FindAllExpired(ctx context.Context, now time.Time) ([]domain.UploadSession, error) {
	args := m.Called(ctx, now)
	return args.Get(0).([]domain.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UploadSessionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

type MockFileTagRepository struct {
	mock.Mock
}

func (m *MockFileTagRepository) DeleteByFileID(ctx context.Context, fileID uuid.UUID) error {
	args := m.Called(ctx, fileID)
	return args.Error(0)
}

func (m *MockFileTagRepository) Create(ctx context.Context, fileID uuid.UUID, tagID uuid.UUID) error {
	args := m.Called(ctx, fileID, tagID)
	return args.Error(0)
}

func (m *MockFileTagRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) ([]domain.FileTag, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0).([]domain.FileTag), args.Error(1)
}

func (m *MockFileTagRepository) CreateMany(ctx context.Context, fileID uuid.UUID, tagIDs []uuid.UUID) (int, error) {
	args := m.Called(ctx, fileID, tagIDs)
	return args.Int(0), args.Error(1)
}

type MockUnitOfWork struct {
	mock.Mock
	tagRepo           *MockTagRepository
	fileRepo          *MockFileRepository
	uploadSessionRepo *MockUploadSessionRepository
	fileTagRepository *MockFileTagRepository
}

func NewMockUnitOfWork() *MockUnitOfWork {
	return &MockUnitOfWork{
		tagRepo:           &MockTagRepository{},
		fileRepo:          &MockFileRepository{},
		uploadSessionRepo: &MockUploadSessionRepository{},
		fileTagRepository: &MockFileTagRepository{},
	}
}

func (m *MockUnitOfWork) FileTagRepo() port.FileTagRepository {
	return m.fileTagRepository
}

func (m *MockUnitOfWork) TagRepo() port.TagRepository {
	return m.tagRepo
}

func (m *MockUnitOfWork) FileRepo() port.FileRepository {
	return m.fileRepo
}

func (m *MockUnitOfWork) UploadSessionRepo() port.UploadSessionRepository {
	return m.uploadSessionRepo
}

func (m *MockUnitOfWork) Execute(ctx context.Context, fn func(uow port.UnitOfWork) error) error {
	args := m.Called(ctx, fn)

	if err := fn(m); err != nil {
		return err
	}

	return args.Error(0)
}

func (m *MockUnitOfWork) GetTagRepoMock() *MockTagRepository {
	return m.tagRepo
}

func (m *MockUnitOfWork) GetFileRepoMock() *MockFileRepository {
	return m.fileRepo
}

func (m *MockUnitOfWork) GetUploadSessionRepoMock() *MockUploadSessionRepository {
	return m.uploadSessionRepo
}

func (m *MockUnitOfWork) GetFileTagRepoMock() *MockFileTagRepository {
	return m.fileTagRepository
}
