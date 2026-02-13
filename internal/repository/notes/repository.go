package notes

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, workspaceID uuid.UUID) ([]model.Note, error) {
	query := `SELECT id, workspace_id, user_id, title, content, created_at, updated_at FROM notes WHERE workspace_id = $1 ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()
	var list []model.Note
	for rows.Next() {
		var n model.Note
		var content sql.NullString
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&n.ID, &n.WorkspaceID, &n.UserID, &n.Title, &content, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		n.CreatedAt = createdAt.Format(time.RFC3339)
		n.UpdatedAt = updatedAt.Format(time.RFC3339)
		if content.Valid {
			n.Content = content.String
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id, workspaceID uuid.UUID) (*model.Note, error) {
	query := `SELECT id, workspace_id, user_id, title, content, created_at, updated_at FROM notes WHERE id = $1 AND workspace_id = $2`
	var n model.Note
	var content sql.NullString
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id, workspaceID).Scan(&n.ID, &n.WorkspaceID, &n.UserID, &n.Title, &content, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	n.CreatedAt = createdAt.Format(time.RFC3339)
	n.UpdatedAt = updatedAt.Format(time.RFC3339)
	if content.Valid {
		n.Content = content.String
	}
	return &n, nil
}

func (r *Repository) Create(ctx context.Context, n *model.Note) error {
	query := `INSERT INTO notes (id, workspace_id, user_id, title, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	wsID, _ := uuid.Parse(n.WorkspaceID)
	userID, _ := uuid.Parse(n.UserID)
	id := uuid.New()
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id, wsID, userID, n.Title, n.Content).Scan(&n.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}
	n.CreatedAt = createdAt.Format(time.RFC3339)
	n.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, n *model.Note) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE notes SET title = $3, content = $4, updated_at = NOW() WHERE id = $1 AND workspace_id = $2`,
		n.ID, n.WorkspaceID, n.Title, n.Content,
	)
	if err != nil {
		return err
	}
	nUpd, _ := res.RowsAffected()
	if nUpd == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM notes WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
