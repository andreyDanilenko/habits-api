package license

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

// HasLicense проверяет, есть ли у пользователя активная лицензия на модуль для данного воркспейса.
// workspaceID == nil при проверке "для любого" (не используется при all_workspaces/single_workspace).
func (r *Repository) HasLicense(ctx context.Context, userID, moduleID uuid.UUID, workspaceID *uuid.UUID) (bool, error) {
	var exists bool
	// Лицензия all_workspaces: одна запись (user_id, module_id, scope=all_workspaces).
	// Лицензия single_workspace: запись с этим workspace_id.
	if workspaceID == nil {
		err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM user_module_licenses
				WHERE user_id = $1 AND module_id = $2 AND status = $3
				AND (scope = $4 OR (scope = $5 AND workspace_id IS NOT NULL))
				AND (expires_at IS NULL OR expires_at > NOW())
			)`,
			userID, moduleID, model.LicenseStatusActive,
			model.LicenseScopeAllWorkspaces, model.LicenseScopeSingleWorkspace,
		).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("has license: %w", err)
		}
		return exists, nil
	}
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_module_licenses
			WHERE user_id = $1 AND module_id = $2 AND status = $3
			AND (
				(scope = $4 AND workspace_id IS NULL) OR
				(scope = $5 AND workspace_id = $6)
			)
			AND (expires_at IS NULL OR expires_at > NOW())
		)`,
		userID, moduleID, model.LicenseStatusActive,
		model.LicenseScopeAllWorkspaces, model.LicenseScopeSingleWorkspace, *workspaceID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has license: %w", err)
	}
	return exists, nil
}

// ListByUserID возвращает все активные лицензии пользователя с кодом модуля.
func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserModuleLicense, error) {
	query := `
		SELECT l.id, l.user_id, l.module_id, m.code, l.scope, l.workspace_id, l.status, l.source, l.expires_at, l.created_at, l.updated_at
		FROM user_module_licenses l
		INNER JOIN modules m ON m.id = l.module_id
		WHERE l.user_id = $1 AND l.status = $2
		AND (l.expires_at IS NULL OR l.expires_at > NOW())
		ORDER BY m.code, l.scope
	`
	rows, err := r.db.QueryContext(ctx, query, userID, model.LicenseStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	var list []model.UserModuleLicense
	for rows.Next() {
		var lic model.UserModuleLicense
		var expiresAt sql.NullTime
		var wsID sql.NullString
		var updatedAt time.Time
		err := rows.Scan(
			&lic.ID, &lic.UserID, &lic.ModuleID, &lic.ModuleCode,
			&lic.Scope, &wsID, &lic.Status, &lic.Source,
			&expiresAt, &lic.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan license: %w", err)
		}
		if wsID.Valid {
			lic.WorkspaceID = &wsID.String
		}
		if expiresAt.Valid {
			t := expiresAt.Time.Format(time.RFC3339)
			lic.ExpiresAt = &t
		}
		lic.UpdatedAt = updatedAt.Format(time.RFC3339)
		list = append(list, lic)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// Create создаёт лицензию (для админа или после оплаты).
func (r *Repository) Create(ctx context.Context, lic *model.UserModuleLicense) error {
	if lic.Status == "" {
		lic.Status = model.LicenseStatusActive
	}
	if lic.Source == "" {
		lic.Source = model.LicenseSourcePurchase
	}
	var workspaceID interface{}
	if lic.WorkspaceID != nil && *lic.WorkspaceID != "" {
		workspaceID = *lic.WorkspaceID
	}
	query := `
		INSERT INTO user_module_licenses (user_id, module_id, scope, workspace_id, status, source, expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		lic.UserID, lic.ModuleID, lic.Scope, workspaceID,
		lic.Status, lic.Source, lic.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create license: %w", err)
	}
	return nil
}
