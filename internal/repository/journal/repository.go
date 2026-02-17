package journal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const listQuery = `SELECT id, workspace_id, user_id, description, mood, date, tags, content_type, metadata, created_at, updated_at
	FROM journal_entries WHERE workspace_id = $1 ORDER BY date DESC, updated_at DESC`

func (r *Repository) List(ctx context.Context, workspaceID uuid.UUID, date *time.Time) ([]model.JournalEntry, error) {
	query := listQuery
	args := []interface{}{workspaceID}
	if date != nil {
		query = `SELECT id, workspace_id, user_id, description, mood, date, tags, content_type, metadata, created_at, updated_at
			FROM journal_entries WHERE workspace_id = $1 AND date = $2 ORDER BY updated_at DESC`
		args = append(args, date.Format("2006-01-02"))
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()
	var list []model.JournalEntry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *entry)
	}
	return list, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id, workspaceID uuid.UUID) (*model.JournalEntry, error) {
	query := `SELECT id, workspace_id, user_id, description, mood, date, tags, content_type, metadata, created_at, updated_at
		FROM journal_entries WHERE id = $1 AND workspace_id = $2`
	row := r.db.QueryRowContext(ctx, query, id, workspaceID)
	entry, err := scanEntryRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *Repository) Create(ctx context.Context, e *model.JournalEntry) error {
	date := e.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	contentType := e.ContentType
	if contentType == "" {
		contentType = "text"
	}
	wsID, _ := uuid.Parse(e.WorkspaceID)
	userID, _ := uuid.Parse(e.UserID)
	id := uuid.New()
	now := time.Now()
	metadataJSON, _ := json.Marshal(e.Metadata)
	if e.Metadata == nil {
		metadataJSON = []byte("{}")
	}
	tags := e.Tags
	if tags == nil {
		tags = []string{}
	}
	query := `INSERT INTO journal_entries (id, workspace_id, user_id, description, mood, date, tags, content_type, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8, $9, $10, $10)
		RETURNING id, created_at, updated_at`
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id, wsID, userID, e.Description, e.Mood, date, pq.Array(tags), contentType, metadataJSON, now).
		Scan(&e.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}
	e.Date = date
	e.ContentType = contentType
	e.CreatedAt = createdAt.Format(time.RFC3339)
	e.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, e *model.JournalEntry) error {
	metadataJSON, _ := json.Marshal(e.Metadata)
	if e.Metadata == nil {
		metadataJSON = []byte("{}")
	}
	tags := e.Tags
	if tags == nil {
		tags = []string{}
	}
	var updatedAt time.Time
	err := r.db.QueryRowContext(ctx, `UPDATE journal_entries SET description = $3, mood = $4, date = $5::date, tags = $6, content_type = $7, metadata = $8, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 RETURNING updated_at`,
		e.ID, e.WorkspaceID, e.Description, e.Mood, e.Date, pq.Array(tags), e.ContentType, metadataJSON).Scan(&updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}
	e.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM journal_entries WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanEntryRow(row *sql.Row) (*model.JournalEntry, error) {
	var e model.JournalEntry
	var mood sql.NullInt32
	var tags pq.StringArray
	var metadataBytes []byte
	var date time.Time
	var createdAt, updatedAt time.Time
	err := row.Scan(&e.ID, &e.WorkspaceID, &e.UserID, &e.Description, &mood, &date, &tags, &e.ContentType, &metadataBytes, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	e.Date = date.Format("2006-01-02")
	e.Tags = tags
	if mood.Valid {
		m := int(mood.Int32)
		e.Mood = &m
	}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &e.Metadata)
	}
	e.CreatedAt = createdAt.Format(time.RFC3339)
	e.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &e, nil
}

func scanEntry(rows *sql.Rows) (*model.JournalEntry, error) {
	var e model.JournalEntry
	var mood sql.NullInt32
	var tags pq.StringArray
	var metadataBytes []byte
	var date time.Time
	var createdAt, updatedAt time.Time
	err := rows.Scan(&e.ID, &e.WorkspaceID, &e.UserID, &e.Description, &mood, &date, &tags, &e.ContentType, &metadataBytes, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	e.Date = date.Format("2006-01-02")
	e.Tags = tags
	if mood.Valid {
		m := int(mood.Int32)
		e.Mood = &m
	}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &e.Metadata)
	}
	e.CreatedAt = createdAt.Format(time.RFC3339)
	e.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &e, nil
}
