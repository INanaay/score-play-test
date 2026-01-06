package postgres

import (
	"context"
	"fmt"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"strings"

	"github.com/google/uuid"
)

type sqlFileTagRepository struct {
	db SQLQuerier
}

// NewFileTagRepository creates sqlFileTagRepository
func NewFileTagRepository(db SQLQuerier) port.FileTagRepository {
	return &sqlFileTagRepository{db: db}
}

// Create inserts a new file tag entry
func (s *sqlFileTagRepository) Create(ctx context.Context, fileID uuid.UUID, tagID uuid.UUID) error {
	query := `INSERT INTO file_metadata_tags (file_id, tag_id) 
              VALUES ($1, $2)`

	_, err := s.db.ExecContext(ctx, query, fileID, tagID)
	if err != nil {
		return fmt.Errorf("error inserting file tag: %w", err)
	}
	return nil
}

// CreateMany creates multiple file-tag associations in batch
func (s *sqlFileTagRepository) CreateMany(ctx context.Context, fileID uuid.UUID, tagIDs []uuid.UUID) (int, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}

	// Remove duplicates
	uniqueTagIDs := make(map[uuid.UUID]bool)
	for _, tagID := range tagIDs {
		uniqueTagIDs[tagID] = true
	}

	tagIDsList := make([]uuid.UUID, 0, len(uniqueTagIDs))
	for tagID := range uniqueTagIDs {
		tagIDsList = append(tagIDsList, tagID)
	}

	// Build placeholders and args
	placeholders := make([]string, len(tagIDsList))
	args := make([]interface{}, len(tagIDsList)*2)

	for i, tagID := range tagIDsList {
		baseIdx := i * 2
		placeholders[i] = fmt.Sprintf("($%d, $%d)", baseIdx+1, baseIdx+2)
		args[baseIdx] = fileID  // file_id
		args[baseIdx+1] = tagID // tag_id
	}

	query := fmt.Sprintf(
		"INSERT INTO file_metadata_tags (file_id, tag_id) VALUES %s ON CONFLICT (file_id, tag_id) DO NOTHING",
		strings.Join(placeholders, ", "),
	)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("error inserting file tags: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// FindByFileID finds all tags for a file
func (s *sqlFileTagRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) ([]domain.FileTag, error) {
	query := `SELECT file_id, tag_id FROM file_metadata_tags WHERE file_id = $1`

	rows, err := s.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("error querying file tags: %w", err)
	}
	defer rows.Close()

	var fileTags []domain.FileTag
	for rows.Next() {
		var dbRelation dbFileTag
		if err := rows.Scan(&dbRelation.FileID, &dbRelation.TagID); err != nil {
			return nil, fmt.Errorf("error scanning file tag: %w", err)
		}
		fileTags = append(fileTags, *dbRelation.ToDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating file tags: %w", err)
	}

	return fileTags, nil
}

// DeleteByFileID removes all tag associations for a given file
func (s *sqlFileTagRepository) DeleteByFileID(ctx context.Context, fileID uuid.UUID) error {
	query := `DELETE FROM file_metadata_tags WHERE file_id = $1`

	_, err := s.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("error deleting file tags: %w", err)
	}
	return nil
}

type dbFileTag struct {
	FileID uuid.UUID `db:"file_id"`
	TagID  uuid.UUID `db:"tag_id"`
}

func (d *dbFileTag) ToDomain() *domain.FileTag {
	return &domain.FileTag{
		FileID: d.FileID,
		TagID:  d.TagID,
	}
}
