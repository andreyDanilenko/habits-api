package model

// PermissionCatalogItem — запись из permission_catalog (для API и UI).
type PermissionCatalogItem struct {
	ID          string `json:"id" db:"id"`
	ModuleCode  string `json:"moduleCode" db:"module_code"`
	EntityType  string `json:"entityType" db:"entity_type"`
	Action      string `json:"action" db:"action"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	IsSystem    bool   `json:"isSystem" db:"is_system"`
	CreatedAt   string `json:"createdAt" db:"created_at"`
}

// PermissionString возвращает строку вида "module:entity:action" для фронта и Casbin obj+act.
func (p *PermissionCatalogItem) PermissionString() string {
	return p.ModuleCode + ":" + p.EntityType + ":" + p.Action
}

// WorkspaceRole — роль в workspace (системная или кастомная).
type WorkspaceRole struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	IsSystem    bool   `json:"isSystem" db:"is_system"`
	CreatedAt   string `json:"createdAt" db:"created_at"`
	UpdatedAt   string `json:"updatedAt" db:"updated_at"`
}

// UserRoleAssignment — назначение роли пользователю в workspace.
type UserRoleAssignment struct {
	ID          string `json:"id" db:"id"`
	UserID      string `json:"userId" db:"user_id"`
	RoleID      string `json:"roleId" db:"role_id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	AssignedBy  string `json:"assignedBy,omitempty" db:"assigned_by"`
	AssignedAt  string `json:"assignedAt" db:"assigned_at"`
}

// UserPermission — индивидуальное право пользователя (минуя роль).
type UserPermission struct {
	ID            string  `json:"id" db:"id"`
	UserID        string  `json:"userId" db:"user_id"`
	WorkspaceID   string  `json:"workspaceId" db:"workspace_id"`
	PermissionID  string  `json:"permissionId" db:"permission_id"`
	GrantedBy     string  `json:"grantedBy,omitempty" db:"granted_by"`
	GrantedAt     string  `json:"grantedAt" db:"granted_at"`
	ExpiresAt     *string `json:"expiresAt,omitempty" db:"expires_at"`
	// Joined from catalog for API
	ModuleCode   string `json:"moduleCode" db:"-"`
	EntityType   string `json:"entityType" db:"-"`
	Action       string `json:"action" db:"-"`
	PermissionStr string `json:"permission" db:"-"`
}

// RoleInheritance описывает отношение наследования ролей в рамках одного workspace.
// childRole наследует права parentRole.
type RoleInheritance struct {
	ID           string `json:"id" db:"id"`
	WorkspaceID  string `json:"workspaceId" db:"workspace_id"`
	ChildRoleID  string `json:"childRoleId" db:"child_role_id"`
	ParentRoleID string `json:"parentRoleId" db:"parent_role_id"`
	CreatedAt    string `json:"createdAt" db:"created_at"`
}
