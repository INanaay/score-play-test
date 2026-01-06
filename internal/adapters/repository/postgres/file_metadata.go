package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"time"

	"github.com/google/uuid"
)

type sqlFileRepository struct {
	db SQLQuerier
}

// NewSqlFileRepository creates sqlFileRepository that implements port.TagRepository
func NewSqlFileRepository(db SQLQuerier) port.FileRepository {
	return &sqlFileRepository{
		db: db,
	}
}

// Create creates new file entry
func (s *sqlFileRepository) Create(ctx context.Context, id uuid.UUID, fileName, mimeType string, mediaType domain.FileType, size int64, status domain.FileStatus, checksum string, storageKey string) error {
	query := `INSERT INTO file_metadata (id, filename, mime_type, file_type, size_bytes, status, checksum, storage_key) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := s.db.ExecContext(ctx, query, id, fileName, mimeType, mediaType, size, status, checksum, storageKey)
	if err != nil {
		return fmt.Errorf("error inserting file metadata: %w", err)
	}
	return nil
}

// UpdateStatus updates status
func (s *sqlFileRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus) error {
	query := `UPDATE file_metadata 
              SET status = $1, updated_at = now()
              WHERE id = $2`

	result, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("error updating file metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrFileMetadataNotFound
	}

	return nil
}

// Delete soft deletes
func (s *sqlFileRepository) Delete(ctx context.Context, id uuid.UUID) error {

	query := `UPDATE file_metadata SET deleted_at = now() WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error updating file metadata: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrFileMetadataNotFound
	}
	return nil
}

// FindById finds by id
func (s *sqlFileRepository) FindById(ctx context.Context, id uuid.UUID) (*domain.FileMetadata, error) {
	query := `SELECT id, filename, mime_type, file_type, size_bytes, storage_key, 
                     checksum, status, created_at, updated_at, deleted_at
              FROM file_metadata
              WHERE id = $1 AND deleted_at IS NULL`

	var dbFile dbFileMetadata
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&dbFile.ID,
		&dbFile.Name,
		&dbFile.MimeType,
		&dbFile.MediaType,
		&dbFile.Size,
		&dbFile.StorageKey,
		&dbFile.Checksum,
		&dbFile.Status,
		&dbFile.CreatedAt,
		&dbFile.UpdatedAt,
		&dbFile.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrFileMetadataNotFound
		}
		return nil, err
	}

	return dbFile.ToDomain(), nil
}

// FindExpired finds expired uploads
func (s *sqlFileRepository) FindExpired(ctx context.Context, expirationTime time.Time) ([]domain.FileMetadata, error) {
	query := `
		SELECT id, filename, mime_type, file_type, size_bytes, storage_key, 
		       checksum, status, created_at, updated_at, deleted_at
		FROM file_metadata
		WHERE status = 'uploading' 
		  AND updated_at < $1 
		  AND deleted_at IS NULL`

	rows, err := s.db.QueryContext(ctx, query, expirationTime)
	if err != nil {
		return nil, fmt.Errorf("error querying expired files: %w", err)
	}
	defer rows.Close()

	var files []domain.FileMetadata
	for rows.Next() {
		var f domain.FileMetadata
		var deletedAt sql.NullTime
		var checksum sql.NullString

		err := rows.Scan(
			&f.ID,
			&f.Filename,
			&f.MimeType,
			&f.MediaType,
			&f.SizeBytes,
			&f.StorageKey,
			&checksum,
			&f.Status,
			&f.CreatedAt,
			&f.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning file metadata: %w", err)
		}

		if checksum.Valid {
			f.Checksum = checksum.String
		}
		if deletedAt.Valid {
			f.DeletedAt = &deletedAt.Time
		}

		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating files: %w", err)
	}

	return files, nil
}

// dbFileMetadata represents file metadata in DB
type dbFileMetadata struct {
	ID         uuid.UUID  `db:"id"`
	Name       string     `db:"filename"`
	MimeType   string     `db:"mime_type"`
	MediaType  string     `db:"file_type"`
	Size       int64      `db:"size_bytes"`
	StorageKey string     `db:"storage_key"`
	Checksum   string     `db:"checksum"`
	Status     string     `db:"status"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

// ToDomain converts to domain.FileStatus
func (f *dbFileMetadata) ToDomain() *domain.FileMetadata {
	return &domain.FileMetadata{
		ID:         f.ID,
		Filename:   f.Name,
		MimeType:   f.MimeType,
		MediaType:  f.MediaType,
		SizeBytes:  f.Size,
		StorageKey: f.StorageKey,
		Checksum:   f.Checksum,
		Status:     domain.FileStatus(f.Status),
		CreatedAt:  f.CreatedAt,
		UpdatedAt:  f.UpdatedAt,
		DeletedAt:  f.DeletedAt,
	}
}
