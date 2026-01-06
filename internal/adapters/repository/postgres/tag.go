package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type sqlTagRepository struct {
	db SQLQuerier
}

// NewSqlTagRepository creates sqlTagRepository that implements port.TagRepository
func NewSqlTagRepository(db SQLQuerier) port.TagRepository {
	return &sqlTagRepository{
		db: db,
	}
}

// Create creates a new tag
func (s *sqlTagRepository) Create(ctx context.Context, name string) error {

	query := `INSERT INTO tags (name) VALUES (LOWER($1))`

	_, err := s.db.ExecContext(ctx, query, name)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return fmt.Errorf("tag %s : %w", name, domain.ErrAlreadyExists)

			}
		}
		return err
	}
	return nil
}

// CreateMany creates multiple tags
func (s *sqlTagRepository) CreateMany(ctx context.Context, tags []string) (int, error) {
	if len(tags) == 0 {
		return 0, nil
	}

	uniqueTags := make(map[string]bool)
	for _, tag := range tags {
		lowerTag := strings.ToLower(tag)
		uniqueTags[lowerTag] = true
	}

	tagsList := make([]string, 0, len(uniqueTags))
	for tag := range uniqueTags {
		tagsList = append(tagsList, tag)
	}

	placeholders := make([]string, len(tagsList))
	args := make([]interface{}, len(tagsList))
	for i, tag := range tagsList {
		placeholders[i] = fmt.Sprintf("(LOWER($%d))", i+1)
		args[i] = tag
	}

	query := fmt.Sprintf(
		"INSERT INTO tags (name) VALUES %s ON CONFLICT DO NOTHING",
		strings.Join(placeholders, ", "),
	)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, domain.ErrAlreadyExists
	}

	return int(rowsAffected), nil
}

// FindByName finds a tag by name
func (s *sqlTagRepository) FindByName(ctx context.Context, name string) (*domain.Tag, error) {
	query := `SELECT id, name, created_at FROM tags WHERE name = LOWER($1)`

	var tagDB dbTag

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&tagDB.ID,
		&tagDB.Name,
		&tagDB.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTagNotFound
		}
		return nil, err
	}

	return tagDB.ToDomain(), nil
}

// FindByNames retrieves multiple tags by their names in a single query
func (s *sqlTagRepository) FindByNames(ctx context.Context, names []string) (map[string]uuid.UUID, error) {
	if len(names) == 0 {
		return make(map[string]uuid.UUID), nil
	}

	lowerNames := make([]string, len(names))
	for i, name := range names {
		lowerNames[i] = strings.ToLower(name)
	}

	placeholders := make([]string, len(lowerNames))
	args := make([]interface{}, len(lowerNames))
	for i, name := range lowerNames {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = name
	}

	query := fmt.Sprintf(
		"SELECT id, name FROM tags WHERE name IN (%s)",
		strings.Join(placeholders, ", "),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying tags: %w", err)
	}
	defer rows.Close()

	result := make(map[string]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("error scanning tag: %w", err)
		}
		result[strings.ToLower(name)] = id
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return result, nil
}

// FindByIDs retrieves multiple tags by their ids in a single query
func (s *sqlTagRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT id, name FROM tags WHERE id IN (%s)",
		strings.Join(placeholders, ", "),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying tags: %w", err)
	}
	defer rows.Close()

	var result []domain.Tag
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("error scanning tag: %w", err)
		}
		result = append(result, domain.Tag{
			ID:   id,
			Name: name,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return result, nil
}

// List retrieves tags with cursor-based pagination sorted by name
func (s *sqlTagRepository) List(ctx context.Context, limit int, marker *string) ([]domain.Tag, *string, error) {
	if limit <= 0 {
		limit = 20 // default limit
	}
	if limit > 100 {
		limit = 100 // max limit
	}

	var query string
	var args []interface{}

	if marker != nil && *marker != "" {
		// Fetch tags after the marker
		// Normalize marker to lowercase for comparison
		lowerMarker := strings.ToLower(*marker)
		query = `
			SELECT id, name, created_at 
			FROM tags 
			WHERE name > $1 
			ORDER BY name ASC 
			LIMIT $2`
		args = []interface{}{lowerMarker, limit + 1}
	} else {
		// Fetch first page
		query = `
			SELECT id, name, created_at 
			FROM tags 
			ORDER BY name ASC 
			LIMIT $1`
		args = []interface{}{limit + 1}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error querying tags: %w", err)
	}
	defer rows.Close()

	tags := make([]domain.Tag, 0, limit)
	for rows.Next() {
		var tagDB dbTag
		if err := rows.Scan(&tagDB.ID, &tagDB.Name, &tagDB.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("error scanning tag: %w", err)
		}
		tags = append(tags, *tagDB.ToDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating tags: %w", err)
	}

	// Check if there are more results
	var nextMarker *string
	if len(tags) > limit {
		// Remove the extra item and set the next marker
		tags = tags[:limit]
		lastName := tags[len(tags)-1].Name
		nextMarker = &lastName
	}

	return tags, nextMarker, nil
}

// TagDB represents a tag in DB
type dbTag struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

// ToDomain converts to domain.Tag
func (t *dbTag) ToDomain() *domain.Tag {
	return &domain.Tag{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
	}
}
