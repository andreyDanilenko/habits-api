package workspace

import (
	"context"
	"database/sql"
	"encoding/json"
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

// ListAll возвращает все workspaces (только для админа).
func (r *Repository) ListAll(ctx context.Context) ([]model.Workspace, error) {
	query := `
		SELECT id, name, description, color, owner_id, created_at, updated_at
		FROM workspaces
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all workspaces: %w", err)
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
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		if description.Valid {
			ws.Description = &description.String
		}
		ws.CreatedAt = createdAt.Format(time.RFC3339)
		ws.UpdatedAt = updatedAt.Format(time.RFC3339)
		workspaces = append(workspaces, ws)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
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

// HasAccess проверяет доступ пользователя к workspace (member в user_workspaces или owner).
func (r *Repository) HasAccess(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workspaces WHERE id = $1 AND owner_id = $2
			UNION ALL
			SELECT 1 FROM user_workspaces WHERE workspace_id = $1 AND user_id = $2
		)
	`, workspaceID, userID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check workspace access: %w", err)
	}
	return ok, nil
}

func (r *Repository) CheckAccess(ctx context.Context, workspaceID, userID uuid.UUID, userRole model.UserRole) (bool, error) {
	if userRole == model.UserRoleAdmin {
		return true, nil
	}
	return r.HasAccess(ctx, workspaceID, userID)
}

// GetModuleByCode возвращает модуль по коду (habits, crm).
func (r *Repository) GetModuleByCode(ctx context.Context, code string) (*model.Module, error) {
	var m model.Module
	var createdAt time.Time
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, code, name, description, is_core, created_at FROM modules WHERE code = $1`,
		code,
	).Scan(&m.ID, &m.Code, &m.Name, &desc, &m.IsCore, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get module by code: %w", err)
	}
	if desc.Valid {
		m.Description = desc.String
	}
	m.CreatedAt = createdAt.Format(time.RFC3339)
	return &m, nil
}

// IsOwner проверяет, что пользователь — владелец воркспейса.
func (r *Repository) IsOwner(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM workspaces WHERE id = $1 AND owner_id = $2)`,
		workspaceID, userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check owner: %w", err)
	}
	return ok, nil
}

// AddWorkspaceModule включает модуль в workspace (INSERT или обновление status на active).
func (r *Repository) AddWorkspaceModule(ctx context.Context, workspaceID, moduleID uuid.UUID) error {
	query := `
		INSERT INTO workspace_modules (workspace_id, module_id, status, activated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (workspace_id, module_id) DO UPDATE SET status = $3
	`
	_, err := r.db.ExecContext(ctx, query, workspaceID, moduleID, model.WorkspaceModuleStatusActive)
	if err != nil {
		return fmt.Errorf("add workspace module: %w", err)
	}
	return nil
}

// SetWorkspaceModuleStatus выставляет статус модуля в workspace (active / disabled).
func (r *Repository) SetWorkspaceModuleStatus(ctx context.Context, workspaceID, moduleID uuid.UUID, status string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE workspace_modules SET status = $3 WHERE workspace_id = $1 AND module_id = $2`,
		workspaceID, moduleID, status,
	)
	if err != nil {
		return fmt.Errorf("set workspace module status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListAllModules возвращает все модули из справочника (для админа — показать все модули в любом воркспейсе).
func (r *Repository) ListAllModules(ctx context.Context) ([]model.Module, error) {
	query := `SELECT id, code, name, description, is_core, created_at FROM modules ORDER BY code`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all modules: %w", err)
	}
	defer rows.Close()
	var list []model.Module
	for rows.Next() {
		var m model.Module
		var createdAt time.Time
		var desc sql.NullString
		err := rows.Scan(&m.ID, &m.Code, &m.Name, &desc, &m.IsCore, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan module: %w", err)
		}
		if desc.Valid {
			m.Description = desc.String
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// ListWorkspaceModules возвращает модули, включённые в workspace (join с modules для code).
func (r *Repository) ListWorkspaceModules(ctx context.Context, workspaceID uuid.UUID) ([]model.WorkspaceModuleInfo, error) {
	query := `
		SELECT wm.id, wm.workspace_id, wm.module_id, m.code, wm.status, wm.settings, wm.created_at
		FROM workspace_modules wm
		INNER JOIN modules m ON m.id = wm.module_id
		WHERE wm.workspace_id = $1
		ORDER BY m.code
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace modules: %w", err)
	}
	defer rows.Close()

	var list []model.WorkspaceModuleInfo
	for rows.Next() {
		var info model.WorkspaceModuleInfo
		var createdAt time.Time
		var settings []byte

		err := rows.Scan(
			&info.ID,
			&info.WorkspaceID,
			&info.ModuleID,
			&info.ModuleName,
			&info.Status,
			&settings,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan workspace module: %w", err)
		}

		info.Enabled = info.Status == model.WorkspaceModuleStatusActive
		info.CreatedAt = createdAt.Format(time.RFC3339)
		info.UpdatedAt = info.CreatedAt
		if settings != nil && len(settings) > 0 {
			_ = json.Unmarshal(settings, &info.Settings)
		}
		list = append(list, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return list, nil
}

// ListAllModulesWithWorkspaceState возвращает все модули системы с флагом enabled для данного workspace.
// Нужно для фронта: показать и доступные, и недоступные (заглушки) с возможностью включить.
func (r *Repository) ListAllModulesWithWorkspaceState(ctx context.Context, workspaceID uuid.UUID) ([]model.WorkspaceModuleInfo, error) {
	query := `
		SELECT m.id AS module_id, m.code, m.name,
		       wm.id AS wm_id, wm.workspace_id, wm.status, wm.settings, wm.created_at
		FROM modules m
		LEFT JOIN workspace_modules wm ON wm.module_id = m.id AND wm.workspace_id = $1
		ORDER BY m.code
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list all modules with workspace state: %w", err)
	}
	defer rows.Close()

	var list []model.WorkspaceModuleInfo
	for rows.Next() {
		var info model.WorkspaceModuleInfo
		var moduleID, code, name string
		var wmID, wmWorkspaceID sql.NullString
		var status sql.NullString
		var settings []byte
		var createdAt sql.NullTime

		err := rows.Scan(&moduleID, &code, &name, &wmID, &wmWorkspaceID, &status, &settings, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		info.ModuleID = moduleID
		info.ModuleName = code
		if wmID.Valid {
			info.ID = wmID.String
		}
		if wmWorkspaceID.Valid {
			info.WorkspaceID = wmWorkspaceID.String
		}
		if status.Valid {
			info.Status = status.String
		} else {
			info.Status = model.WorkspaceModuleStatusDisabled
		}
		info.Enabled = info.Status == model.WorkspaceModuleStatusActive
		if createdAt.Valid {
			info.CreatedAt = createdAt.Time.Format(time.RFC3339)
			info.UpdatedAt = info.CreatedAt
		}
		if settings != nil && len(settings) > 0 {
			_ = json.Unmarshal(settings, &info.Settings)
		}
		list = append(list, info)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
