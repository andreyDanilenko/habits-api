package permission

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrPermissionNotFound = errors.New("permission not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListCatalog возвращает все записи из permission_catalog.
func (r *Repository) ListCatalog(ctx context.Context) ([]model.PermissionCatalogItem, error) {
	query := `
		SELECT id, module_code, entity_type, action, name, COALESCE(description,''), is_system, created_at
		FROM permission_catalog
		ORDER BY module_code, entity_type, action
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list catalog: %w", err)
	}
	defer rows.Close()
	var list []model.PermissionCatalogItem
	for rows.Next() {
		var p model.PermissionCatalogItem
		var createdAt time.Time
		err := rows.Scan(&p.ID, &p.ModuleCode, &p.EntityType, &p.Action, &p.Name, &p.Description, &p.IsSystem, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan catalog: %w", err)
		}
		p.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// GetCatalogByID возвращает одну запись каталога по ID.
func (r *Repository) GetCatalogByID(ctx context.Context, id string) (*model.PermissionCatalogItem, error) {
	query := `
		SELECT id, module_code, entity_type, action, name, COALESCE(description,''), is_system, created_at
		FROM permission_catalog WHERE id = $1
	`
	var p model.PermissionCatalogItem
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.ModuleCode, &p.EntityType, &p.Action, &p.Name, &p.Description, &p.IsSystem, &createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get catalog by id: %w", err)
	}
	p.CreatedAt = createdAt.Format(time.RFC3339)
	return &p, nil
}

// ListRolesByWorkspace возвращает все роли workspace.
func (r *Repository) ListRolesByWorkspace(ctx context.Context, workspaceID string) ([]model.WorkspaceRole, error) {
	query := `
		SELECT id, workspace_id, name, COALESCE(description,''), is_system, created_at, updated_at
		FROM workspace_roles WHERE workspace_id = $1 ORDER BY is_system DESC, name
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()
	var list []model.WorkspaceRole
	for rows.Next() {
		var wr model.WorkspaceRole
		var createdAt, updatedAt time.Time
		err := rows.Scan(&wr.ID, &wr.WorkspaceID, &wr.Name, &wr.Description, &wr.IsSystem, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		wr.CreatedAt = createdAt.Format(time.RFC3339)
		wr.UpdatedAt = updatedAt.Format(time.RFC3339)
		list = append(list, wr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// GetRoleByID возвращает роль по ID.
func (r *Repository) GetRoleByID(ctx context.Context, roleID string) (*model.WorkspaceRole, error) {
	query := `
		SELECT id, workspace_id, name, COALESCE(description,''), is_system, created_at, updated_at
		FROM workspace_roles WHERE id = $1
	`
	var wr model.WorkspaceRole
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, roleID).Scan(
		&wr.ID, &wr.WorkspaceID, &wr.Name, &wr.Description, &wr.IsSystem, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get role: %w", err)
	}
	wr.CreatedAt = createdAt.Format(time.RFC3339)
	wr.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &wr, nil
}

// GetRoleByName возвращает роль workspace по имени.
func (r *Repository) GetRoleByName(ctx context.Context, workspaceID, name string) (*model.WorkspaceRole, error) {
	query := `
		SELECT id, workspace_id, name, COALESCE(description,''), is_system, created_at, updated_at
		FROM workspace_roles WHERE workspace_id = $1 AND name = $2
	`
	var wr model.WorkspaceRole
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, workspaceID, name).Scan(
		&wr.ID, &wr.WorkspaceID, &wr.Name, &wr.Description, &wr.IsSystem, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	wr.CreatedAt = createdAt.Format(time.RFC3339)
	wr.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &wr, nil
}

// CreateRole создаёт роль в workspace.
func (r *Repository) CreateRole(ctx context.Context, role *model.WorkspaceRole) error {
	id := uuid.New().String()
	now := time.Now()
	query := `
		INSERT INTO workspace_roles (id, workspace_id, name, description, is_system, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
	`
	_, err := r.db.ExecContext(ctx, query, id, role.WorkspaceID, role.Name, role.Description, role.IsSystem, now)
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}
	role.ID = id
	role.CreatedAt = now.Format(time.RFC3339)
	role.UpdatedAt = role.CreatedAt
	return nil
}

// UpdateRole обновляет имя и описание роли.
func (r *Repository) UpdateRole(ctx context.Context, roleID, name, description string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE workspace_roles SET name = $2, description = $3, updated_at = NOW() WHERE id = $1
	`, roleID, name, description)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// DeleteRole удаляет роль (только кастомную; проверка is_system — в сервисе).
func (r *Repository) DeleteRole(ctx context.Context, roleID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM workspace_roles WHERE id = $1`, roleID)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// ListUserRoleAssignments возвращает все назначения ролей пользователю в workspace.
func (r *Repository) ListUserRoleAssignments(ctx context.Context, userID, workspaceID string) ([]model.UserRoleAssignment, error) {
	query := `
		SELECT ura.id, ura.user_id, ura.role_id, ura.workspace_id, ura.assigned_by, ura.assigned_at
		FROM user_role_assignments ura
		WHERE ura.user_id = $1 AND ura.workspace_id = $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list user role assignments: %w", err)
	}
	defer rows.Close()
	var list []model.UserRoleAssignment
	for rows.Next() {
		var a model.UserRoleAssignment
		var assignedAt time.Time
		var assignedBy sql.NullString
		err := rows.Scan(&a.ID, &a.UserID, &a.RoleID, &a.WorkspaceID, &assignedBy, &assignedAt)
		if err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		if assignedBy.Valid {
			a.AssignedBy = assignedBy.String
		}
		a.AssignedAt = assignedAt.Format(time.RFC3339)
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// CreateUserRoleAssignment создаёт назначение роли пользователю.
func (r *Repository) CreateUserRoleAssignment(ctx context.Context, a *model.UserRoleAssignment) error {
	id := uuid.New().String()
	query := `
		INSERT INTO user_role_assignments (id, user_id, role_id, workspace_id, assigned_by, assigned_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	assignedBy := sql.NullString{}
	if a.AssignedBy != "" {
		assignedBy = sql.NullString{String: a.AssignedBy, Valid: true}
	}
	_, err := r.db.ExecContext(ctx, query, id, a.UserID, a.RoleID, a.WorkspaceID, assignedBy)
	if err != nil {
		return fmt.Errorf("create user role assignment: %w", err)
	}
	a.ID = id
	return nil
}

// DeleteUserRoleAssignment удаляет назначение.
func (r *Repository) DeleteUserRoleAssignment(ctx context.Context, userID, roleID, workspaceID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM user_role_assignments WHERE user_id = $1 AND role_id = $2 AND workspace_id = $3
	`, userID, roleID, workspaceID)
	if err != nil {
		return fmt.Errorf("delete user role assignment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRoleNotFound
	}
	return nil
}

// ListUserPermissions возвращает индивидуальные права пользователя в workspace (с данными каталога).
func (r *Repository) ListUserPermissions(ctx context.Context, userID, workspaceID string) ([]model.UserPermission, error) {
	query := `
		SELECT up.id, up.user_id, up.workspace_id, up.permission_id, up.granted_by, up.granted_at, up.expires_at,
		       pc.module_code, pc.entity_type, pc.action
		FROM user_permissions up
		JOIN permission_catalog pc ON pc.id = up.permission_id
		WHERE up.user_id = $1 AND up.workspace_id = $2
		  AND (up.expires_at IS NULL OR up.expires_at > NOW())
	`
	rows, err := r.db.QueryContext(ctx, query, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list user permissions: %w", err)
	}
	defer rows.Close()
	var list []model.UserPermission
	for rows.Next() {
		var up model.UserPermission
		var grantedAt time.Time
		var grantedBy sql.NullString
		var expiresAt sql.NullTime
		err := rows.Scan(&up.ID, &up.UserID, &up.WorkspaceID, &up.PermissionID, &grantedBy, &grantedAt, &expiresAt,
			&up.ModuleCode, &up.EntityType, &up.Action)
		if err != nil {
			return nil, fmt.Errorf("scan user permission: %w", err)
		}
		if grantedBy.Valid {
			up.GrantedBy = grantedBy.String
		}
		up.GrantedAt = grantedAt.Format(time.RFC3339)
		if expiresAt.Valid {
			t := expiresAt.Time.Format(time.RFC3339)
			up.ExpiresAt = &t
		}
		up.PermissionStr = up.ModuleCode + ":" + up.EntityType + ":" + up.Action
		list = append(list, up)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// CreateUserPermission выдаёт индивидуальное право.
func (r *Repository) CreateUserPermission(ctx context.Context, up *model.UserPermission) error {
	id := uuid.New().String()
	grantedBy := sql.NullString{}
	if up.GrantedBy != "" {
		grantedBy = sql.NullString{String: up.GrantedBy, Valid: true}
	}
	var expiresAt interface{}
	if up.ExpiresAt != nil && *up.ExpiresAt != "" {
		expiresAt = up.ExpiresAt
	}
	query := `
		INSERT INTO user_permissions (id, user_id, workspace_id, permission_id, granted_by, granted_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
	`
	_, err := r.db.ExecContext(ctx, query, id, up.UserID, up.WorkspaceID, up.PermissionID, grantedBy, expiresAt)
	if err != nil {
		return fmt.Errorf("create user permission: %w", err)
	}
	up.ID = id
	return nil
}

// DeleteUserPermission отзывает индивидуальное право.
func (r *Repository) DeleteUserPermission(ctx context.Context, userID, workspaceID, permissionID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM user_permissions WHERE user_id = $1 AND workspace_id = $2 AND permission_id = $3
	`, userID, workspaceID, permissionID)
	if err != nil {
		return fmt.Errorf("delete user permission: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrPermissionNotFound
	}
	return nil
}

// CountAssignmentsByRole возвращает число назначений для роли (для проверки перед удалением).
func (r *Repository) CountAssignmentsByRole(ctx context.Context, roleID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_role_assignments WHERE role_id = $1`, roleID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count assignments: %w", err)
	}
	return n, nil
}

// ListDistinctWorkspaceIDs возвращает уникальные workspace_id из workspace_roles (для загрузки системных политик).
func (r *Repository) ListDistinctWorkspaceIDs(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT workspace_id FROM workspace_roles ORDER BY workspace_id`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list workspace ids: %w", err)
	}
	defer rows.Close()
	var list []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		list = append(list, id)
	}
	return list, rows.Err()
}

// ListAllUserRoleAssignments возвращает все назначения (для синхронизации с Casbin).
func (r *Repository) ListAllUserRoleAssignments(ctx context.Context) ([]model.UserRoleAssignment, error) {
	query := `
		SELECT ura.id, ura.user_id, ura.role_id, ura.workspace_id, ura.assigned_by, ura.assigned_at
		FROM user_role_assignments ura
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all assignments: %w", err)
	}
	defer rows.Close()
	var list []model.UserRoleAssignment
	for rows.Next() {
		var a model.UserRoleAssignment
		var assignedAt time.Time
		var assignedBy sql.NullString
		err := rows.Scan(&a.ID, &a.UserID, &a.RoleID, &a.WorkspaceID, &assignedBy, &assignedAt)
		if err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		if assignedBy.Valid {
			a.AssignedBy = assignedBy.String
		}
		a.AssignedAt = assignedAt.Format(time.RFC3339)
		list = append(list, a)
	}
	return list, rows.Err()
}

// ListRoleInheritanceByWorkspace возвращает все отношения наследования ролей в workspace.
func (r *Repository) ListRoleInheritanceByWorkspace(ctx context.Context, workspaceID string) ([]model.RoleInheritance, error) {
	query := `
		SELECT id, workspace_id, child_role_id, parent_role_id, created_at
		FROM role_inheritance
		WHERE workspace_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list role inheritance: %w", err)
	}
	defer rows.Close()
	var list []model.RoleInheritance
	for rows.Next() {
		var ri model.RoleInheritance
		var createdAt time.Time
		if err := rows.Scan(&ri.ID, &ri.WorkspaceID, &ri.ChildRoleID, &ri.ParentRoleID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan role inheritance: %w", err)
		}
		ri.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, ri)
	}
	return list, rows.Err()
}

// CreateRoleInheritance создаёт отношение наследования ролей.
func (r *Repository) CreateRoleInheritance(ctx context.Context, workspaceID, childRoleID, parentRoleID string) error {
	id := uuid.New().String()
	query := `
		INSERT INTO role_inheritance (id, workspace_id, child_role_id, parent_role_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.ExecContext(ctx, query, id, workspaceID, childRoleID, parentRoleID)
	if err != nil {
		return fmt.Errorf("create role inheritance: %w", err)
	}
	return nil
}

// DeleteRoleInheritance удаляет отношение наследования ролей.
func (r *Repository) DeleteRoleInheritance(ctx context.Context, workspaceID, childRoleID, parentRoleID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM role_inheritance
		WHERE workspace_id = $1 AND child_role_id = $2 AND parent_role_id = $3
	`, workspaceID, childRoleID, parentRoleID)
	if err != nil {
		return fmt.Errorf("delete role inheritance: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRoleNotFound
	}
	return nil
}
