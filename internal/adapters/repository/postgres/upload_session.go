package postgres

import (
	"context"
	"database/sql"
	"errors"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"time"

	"github.com/google/uuid"
)

type sqlUploadSessionRepository struct {
	db SQLQuerier
}

// NewSQLUploadSessionRepository Creates a new sqlUploadSessionRepository
func NewSQLUploadSessionRepository(db SQLQuerier) port.UploadSessionRepository {
	return &sqlUploadSessionRepository{db: db}
}

// Create creates an upload session
func (s *sqlUploadSessionRepository) Create(ctx context.Context, session domain.UploadSession) error {
	query := `
		INSERT INTO upload_session (
			id, file_id, provider_upload_id, part_size, expires_at, status
		) VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.FileID,
		session.ProviderUploadID,
		session.PartSize,
		session.ExpiresAt,
		session.Status,
	)
	if err != nil {
		return err
	}
	return nil
}

// UpdateExpiresAt updates expires at
func (s *sqlUploadSessionRepository) UpdateExpiresAt(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	query := `UPDATE upload_session SET expires_at = $1, updated_at = now() WHERE id = $2 AND status = 'open'`

	result, err := s.db.ExecContext(ctx, query, expiresAt, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}

func (s *sqlUploadSessionRepository) FindByIDAndActive(ctx context.Context, id uuid.UUID) (*domain.UploadSession, error) {
	query := `
		SELECT id, file_id, provider_upload_id, part_size, expires_at, status, created_at, updated_at
		FROM upload_session 
		WHERE id = $1 AND status = 'open'`

	var row dbUploadSession
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.FileID,
		&row.ProviderUploadID,
		&row.PartSize,
		&row.ExpiresAt,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}

	return row.ToDomain(), nil
}

// UpdateStatusByFileID updates session status by file ID
func (s *sqlUploadSessionRepository) UpdateStatusByFileID(ctx context.Context, fileID uuid.UUID, status domain.UploadSessionStatus) error {
	query := `UPDATE upload_session SET status = $1, updated_at = now() WHERE file_id = $2`

	result, err := s.db.ExecContext(ctx, query, status, fileID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}

func (s *sqlUploadSessionRepository) FindAllExpired(ctx context.Context, now time.Time) ([]domain.UploadSession, error) {
	query := `
		SELECT id, file_id, provider_upload_id, part_size, expires_at, status, created_at, updated_at
		FROM upload_session 
		WHERE status = 'open' AND expires_at < $1`

	rows, err := s.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.UploadSession
	for rows.Next() {
		var row dbUploadSession
		if err := rows.Scan(
			&row.ID,
			&row.FileID,
			&row.ProviderUploadID,
			&row.PartSize,
			&row.ExpiresAt,
			&row.Status,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, *row.ToDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (s *sqlUploadSessionRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) (*domain.UploadSession, error) {
	query := `
		SELECT id, file_id, provider_upload_id, part_size, expires_at, status, created_at, updated_at
		FROM upload_session 
		WHERE file_id = $1 AND status = 'open'`

	var row dbUploadSession
	err := s.db.QueryRowContext(ctx, query, fileID).Scan(
		&row.ID,
		&row.FileID,
		&row.ProviderUploadID,
		&row.PartSize,
		&row.ExpiresAt,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}

	return row.ToDomain(), nil
}

func (s *sqlUploadSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.UploadSession, error) {
	query := `
		SELECT id, file_id, provider_upload_id, part_size, expires_at, status, created_at, updated_at
		FROM upload_session 
		WHERE id = $1`

	var row dbUploadSession
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.FileID,
		&row.ProviderUploadID,
		&row.PartSize,
		&row.ExpiresAt,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}

	return row.ToDomain(), nil
}

// UpdateStatus updates status
func (s *sqlUploadSessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UploadSessionStatus) error {
	query := `UPDATE upload_session SET status = $1, updated_at = now() WHERE id = $2`

	result, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}

type dbUploadSession struct {
	ID               uuid.UUID `db:"id"`
	FileID           uuid.UUID `db:"file_id"`
	ProviderUploadID string    `db:"provider_upload_id"`
	PartSize         int       `db:"part_size"`
	ExpiresAt        time.Time `db:"expires_at"`
	Status           string    `db:"status"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// ToDomain converts db obj to domain
func (s *dbUploadSession) ToDomain() *domain.UploadSession {
	return &domain.UploadSession{
		ID:               s.ID,
		FileID:           s.FileID,
		ProviderUploadID: s.ProviderUploadID,
		PartSize:         s.PartSize,
		ExpiresAt:        s.ExpiresAt,
		Status:           domain.UploadSessionStatus(s.Status),
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}
