package user_preferences

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("user preferences not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetCurrentWorkspace(ctx context.Context, userID uuid.UUID) (string, error) {
	query := `
		SELECT current_workspace_id::text FROM user_preferences
		WHERE user_id = $1 AND current_workspace_id IS NOT NULL
	`
	var wsID sql.NullString
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&wsID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get current workspace: %w", err)
	}
	if !wsID.Valid {
		return "", nil
	}
	return wsID.String, nil
}

func (r *Repository) SetCurrentWorkspace(ctx context.Context, userID uuid.UUID, workspaceID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace id: %w", err)
	}
	now := time.Now()
	query := `
		INSERT INTO user_preferences (id, user_id, current_workspace_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			current_workspace_id = EXCLUDED.current_workspace_id,
			updated_at = EXCLUDED.updated_at
	`
	_, err = r.db.ExecContext(ctx, query, userID, wsID, now)
	if err != nil {
		return fmt.Errorf("set current workspace: %w", err)
	}
	return nil
}

func (r *Repository) UnsetCurrentWorkspace(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE user_preferences SET current_workspace_id = NULL, updated_at = $1
		WHERE user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("unset current workspace: %w", err)
	}
	return nil
}
