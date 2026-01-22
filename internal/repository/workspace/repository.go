package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]model.Workspace, error) {
	query := `
		SELECT w.id, w.name, w.description, w.color, w.owner_id, w.created_at, w.updated_at
		FROM workspaces w
		INNER JOIN user_workspaces uw ON w.id = uw.workspace_id
		WHERE uw.user_id = $1
		ORDER BY w.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []model.Workspace
	for rows.Next() {
		var ws model.Workspace
		var createdAt, updatedAt time.Time
		var description sql.NullString

		err := rows.Scan(
			&ws.ID,
			&ws.Name,
			&description,
			&ws.Color,
			&ws.OwnerID,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace: %w", err)
		}

		if description.Valid {
			ws.Description = &description.String
		}
		ws.CreatedAt = createdAt.Format(time.RFC3339)
		ws.UpdatedAt = updatedAt.Format(time.RFC3339)
		workspaces = append(workspaces, ws)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return workspaces, nil
}

func (r *Repository) Create(ctx context.Context, dto model.CreateWorkspaceDto, ownerID uuid.UUID) (*model.Workspace, error) {
	workspaceID := uuid.New()
	now := time.Now()
	color := "#3B82F6"
	if dto.Color != nil {
		color = *dto.Color
	}

	query := `
		INSERT INTO workspaces (id, name, description, color, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, description, color, owner_id, created_at, updated_at
	`

	var ws model.Workspace
	var createdAt, updatedAt time.Time
	var description sql.NullString

	err := r.db.QueryRowContext(ctx, query,
		workspaceID,
		dto.Name,
		dto.Description,
		color,
		ownerID,
		now,
		now,
	).Scan(
		&ws.ID,
		&ws.Name,
		&description,
		&ws.Color,
		&ws.OwnerID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	if description.Valid {
		ws.Description = &description.String
	}
	ws.CreatedAt = createdAt.Format(time.RFC3339)
	ws.UpdatedAt = updatedAt.Format(time.RFC3339)

	// Добавляем владельца в user_workspaces
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO user_workspaces (user_id, workspace_id, role) VALUES ($1, $2, 'OWNER')",
		ownerID, workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add owner to workspace: %w", err)
	}

	return &ws, nil
}

func (r *Repository) Get(ctx context.Context, workspaceID uuid.UUID) (*model.Workspace, error) {
	query := `
		SELECT id, name, description, color, owner_id, created_at, updated_at
		FROM workspaces
		WHERE id = $1
	`

	var ws model.Workspace
	var createdAt, updatedAt time.Time
	var description sql.NullString

	err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(
		&ws.ID,
		&ws.Name,
		&description,
		&ws.Color,
		&ws.OwnerID,
		&createdAt,
		&updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	if description.Valid {
		ws.Description = &description.String
	}
	ws.CreatedAt = createdAt.Format(time.RFC3339)
	ws.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &ws, nil
}

func (r *Repository) Update(ctx context.Context, workspaceID uuid.UUID, dto model.UpdateWorkspaceDto) (*model.Workspace, error) {
	now := time.Now()
	query := `
		UPDATE workspaces SET
			name = COALESCE($1, name),
			description = COALESCE($2, description),
			color = COALESCE($3, color),
			updated_at = $4
		WHERE id = $5
		RETURNING id, name, description, color, owner_id, created_at, updated_at
	`

	var ws model.Workspace
	var createdAt, updatedAt time.Time
	var description sql.NullString

	err := r.db.QueryRowContext(ctx, query,
		dto.Name,
		dto.Description,
		dto.Color,
		now,
		workspaceID,
	).Scan(
		&ws.ID,
		&ws.Name,
		&description,
		&ws.Color,
		&ws.OwnerID,
		&createdAt,
		&updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	if description.Valid {
		ws.Description = &description.String
	}
	ws.CreatedAt = createdAt.Format(time.RFC3339)
	ws.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &ws, nil
}

func (r *Repository) Delete(ctx context.Context, workspaceID uuid.UUID) error {
	query := `DELETE FROM workspaces WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) CheckAccess(ctx context.Context, workspaceID, userID uuid.UUID, userRole model.UserRole) (bool, error) {
	// Админ имеет доступ ко всем workspace
	if userRole == model.UserRoleAdmin {
		return true, nil
	}

	// Проверяем, является ли пользователь владельцем
	var ownerID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		"SELECT owner_id FROM workspaces WHERE id = $1",
		workspaceID,
	).Scan(&ownerID)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check workspace owner: %w", err)
	}

	return ownerID == userID, nil
}
