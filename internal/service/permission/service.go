package permission

import (
	"context"
	"errors"
	"strings"

	"backend/internal/model"
	permRepo "backend/internal/repository/permission"

	casbin "github.com/casbin/casbin/v3"
)

var (
	ErrRoleSystem      = errors.New("cannot modify or delete system role")
	ErrRoleHasUsers    = errors.New("cannot delete role with assigned users")
	ErrInvalidPermForm = errors.New("permission must be format module:entity:action")
)

type Service struct {
	repo     *permRepo.Repository
	enforcer *casbin.Enforcer
}

func NewService(repo *permRepo.Repository, enforcer *casbin.Enforcer) *Service {
	return &Service{repo: repo, enforcer: enforcer}
}

// GetCatalog возвращает каталог прав для UI.
func (s *Service) GetCatalog(ctx context.Context) ([]model.PermissionCatalogItem, error) {
	return s.repo.ListCatalog(ctx)
}

// ListRoles возвращает все роли workspace.
func (s *Service) ListRoles(ctx context.Context, workspaceID string) ([]model.WorkspaceRole, error) {
	return s.repo.ListRolesByWorkspace(ctx, workspaceID)
}

// GetRole возвращает роль по ID.
func (s *Service) GetRole(ctx context.Context, roleID string) (*model.WorkspaceRole, error) {
	return s.repo.GetRoleByID(ctx, roleID)
}

// GetRolePermissions возвращает права роли в формате "module:entity:action" (из Casbin).
func (s *Service) GetRolePermissions(ctx context.Context, roleID string) ([]string, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || role == nil {
		return nil, permRepo.ErrRoleNotFound
	}
	sub := "role:" + role.Name
	policies, err := s.enforcer.GetFilteredPolicy(0, sub)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range policies {
		if len(p) >= 4 {
			obj, act := p[2], p[3]
			// dom is p[1]; we have obj and act; format "module:entity:action" — obj is already "module:entity"
			out = append(out, obj+":"+act)
		}
	}
	return out, nil
}

// CreateRole создаёт кастомную роль и добавляет политики в Casbin.
// При сбое сохранения политик откатывает создание роли в БД.
func (s *Service) CreateRole(ctx context.Context, workspaceID, name, description string, permissions []string, createdBy string) (*model.WorkspaceRole, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("role name is required")
	}
	existing, _ := s.repo.GetRoleByName(ctx, workspaceID, name)
	if existing != nil {
		return nil, errors.New("role with this name already exists in workspace")
	}
	role := &model.WorkspaceRole{
		WorkspaceID: workspaceID,
		Name:        name,
		Description: description,
		IsSystem:    false,
	}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	for _, perm := range permissions {
		obj, act, err := parsePermission(perm)
		if err != nil {
			continue
		}
		if _, err := s.enforcer.AddPolicy("role:"+role.Name, workspaceID, obj, act); err != nil {
			_ = s.repo.DeleteRole(ctx, role.ID)
			return nil, err
		}
	}
	if err := s.enforcer.SavePolicy(); err != nil {
		_ = s.repo.DeleteRole(ctx, role.ID)
		return nil, err
	}
	return role, nil
}

// UpdateRole обновляет роль и перезаписывает политики в Casbin (только кастомная роль).
// При смене имени роли политики удаляются по старому имени и добавляются по новому.
func (s *Service) UpdateRole(ctx context.Context, roleID, name, description string, permissions []string) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || role == nil {
		return permRepo.ErrRoleNotFound
	}
	if role.IsSystem {
		return ErrRoleSystem
	}
	oldRoleName := role.Name
	effectiveName := oldRoleName
	if name != "" {
		effectiveName = strings.TrimSpace(name)
		if effectiveName != oldRoleName {
			existing, _ := s.repo.GetRoleByName(ctx, role.WorkspaceID, effectiveName)
			if existing != nil {
				return errors.New("role with this name already exists in workspace")
			}
		}
		if err := s.repo.UpdateRole(ctx, roleID, effectiveName, description); err != nil {
			return err
		}
		role.Name = effectiveName
	} else if description != "" {
		_ = s.repo.UpdateRole(ctx, roleID, role.Name, description)
	}
	// Удаляем политики по старому имени роли (в Casbin они ещё под старым именем)
	oldPolicies, _ := s.enforcer.GetFilteredPolicy(0, "role:"+oldRoleName)
	for _, p := range oldPolicies {
		if len(p) >= 4 && p[1] == role.WorkspaceID {
			_, _ = s.enforcer.RemovePolicy(p[0], p[1], p[2], p[3])
		}
	}
	for _, perm := range permissions {
		obj, act, err := parsePermission(perm)
		if err != nil {
			continue
		}
		_, _ = s.enforcer.AddPolicy("role:"+effectiveName, role.WorkspaceID, obj, act)
	}
	return s.enforcer.SavePolicy()
}

// DeleteRole удаляет роль (только кастомную, без назначений).
// Сначала удаляет политики в Casbin, затем запись в БД.
func (s *Service) DeleteRole(ctx context.Context, roleID string) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || role == nil {
		return permRepo.ErrRoleNotFound
	}
	if role.IsSystem {
		return ErrRoleSystem
	}
	n, err := s.repo.CountAssignmentsByRole(ctx, roleID)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrRoleHasUsers
	}
	oldPolicies, _ := s.enforcer.GetFilteredPolicy(0, "role:"+role.Name)
	for _, p := range oldPolicies {
		if len(p) >= 4 {
			_, _ = s.enforcer.RemovePolicy(p[0], p[1], p[2], p[3])
		}
	}
	if err := s.enforcer.SavePolicy(); err != nil {
		return err
	}
	return s.repo.DeleteRole(ctx, roleID)
}

// AssignRoleByName назначает роль пользователю по имени (OWNER, ADMIN, MEMBER, GUEST).
func (s *Service) AssignRoleByName(ctx context.Context, userID, workspaceID, roleName, assignedBy string) error {
	role, err := s.repo.GetRoleByName(ctx, workspaceID, roleName)
	if err != nil || role == nil {
		return permRepo.ErrRoleNotFound
	}
	return s.AssignRole(ctx, userID, role.ID, workspaceID, assignedBy)
}

// AssignRole назначает роль пользователю и добавляет групповую политику в Casbin.
// При сбое сохранения политик откатывает назначение в БД.
func (s *Service) AssignRole(ctx context.Context, userID, roleID, workspaceID, assignedBy string) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || role == nil {
		return permRepo.ErrRoleNotFound
	}
	if role.WorkspaceID != workspaceID {
		return errors.New("role does not belong to this workspace")
	}
	a := &model.UserRoleAssignment{
		UserID: userID, RoleID: roleID, WorkspaceID: workspaceID, AssignedBy: assignedBy,
	}
	if err := s.repo.CreateUserRoleAssignment(ctx, a); err != nil {
		return err
	}
	if _, err := s.enforcer.AddGroupingPolicy("user:"+userID, "role:"+role.Name, workspaceID); err != nil {
		_ = s.repo.DeleteUserRoleAssignment(ctx, userID, roleID, workspaceID)
		return err
	}
	if err := s.enforcer.SavePolicy(); err != nil {
		_, _ = s.enforcer.RemoveGroupingPolicy("user:"+userID, "role:"+role.Name, workspaceID)
		_ = s.repo.DeleteUserRoleAssignment(ctx, userID, roleID, workspaceID)
		return err
	}
	return nil
}

// RemoveRole снимает роль с пользователя и удаляет групповую политику.
func (s *Service) RemoveRole(ctx context.Context, userID, roleID, workspaceID string) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || role == nil {
		return permRepo.ErrRoleNotFound
	}
	if err := s.repo.DeleteUserRoleAssignment(ctx, userID, roleID, workspaceID); err != nil {
		return err
	}
	_, _ = s.enforcer.RemoveGroupingPolicy("user:"+userID, "role:"+role.Name, workspaceID)
	return s.enforcer.SavePolicy()
}

// GetUserRoles возвращает имена ролей пользователя в workspace.
func (s *Service) GetUserRoles(ctx context.Context, userID, workspaceID string) ([]string, error) {
	assignments, err := s.repo.ListUserRoleAssignments(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, a := range assignments {
		role, _ := s.repo.GetRoleByID(ctx, a.RoleID)
		if role != nil {
			names = append(names, role.Name)
		}
	}
	return names, nil
}

// GetUserRolesFull возвращает полные роли (для API списка ролей пользователя).
func (s *Service) GetUserRolesFull(ctx context.Context, userID, workspaceID string) ([]model.WorkspaceRole, error) {
	assignments, err := s.repo.ListUserRoleAssignments(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	var roles []model.WorkspaceRole
	for _, a := range assignments {
		role, _ := s.repo.GetRoleByID(ctx, a.RoleID)
		if role != nil {
			roles = append(roles, *role)
		}
	}
	return roles, nil
}

// GrantPermission выдаёт индивидуальное право.
func (s *Service) GrantPermission(ctx context.Context, userID, workspaceID, permissionID string, grantedBy string, expiresAt *string) error {
	item, err := s.repo.GetCatalogByID(ctx, permissionID)
	if err != nil || item == nil {
		return permRepo.ErrPermissionNotFound
	}
	_ = item
	up := &model.UserPermission{
		UserID:       userID,
		WorkspaceID:  workspaceID,
		PermissionID: permissionID,
		GrantedBy:    grantedBy,
		ExpiresAt:    expiresAt,
	}
	return s.repo.CreateUserPermission(ctx, up)
}

// RevokePermission отзывает индивидуальное право.
func (s *Service) RevokePermission(ctx context.Context, userID, workspaceID, permissionID string) error {
	return s.repo.DeleteUserPermission(ctx, userID, workspaceID, permissionID)
}

// GetUserPermissions возвращает индивидуальные права пользователя (строки "module:entity:action").
func (s *Service) GetUserPermissions(ctx context.Context, userID, workspaceID string) ([]model.UserPermission, error) {
	return s.repo.ListUserPermissions(ctx, userID, workspaceID)
}

// GetEffectivePermissions возвращает все права пользователя в workspace (из ролей + индивидуальные) для UI.
func (s *Service) GetEffectivePermissions(ctx context.Context, userID, workspaceID string) ([]string, error) {
	seen := make(map[string]bool)
	// From roles (Casbin policies for user's roles in this domain)
	roles, err := s.GetUserRoles(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, roleName := range roles {
		policies, _ := s.enforcer.GetFilteredPolicy(0, "role:"+roleName)
		for _, p := range policies {
			if len(p) >= 4 && p[1] == workspaceID {
				perm := p[2] + ":" + p[3]
				seen[perm] = true
			}
		}
	}
	// Individual permissions
	list, err := s.repo.ListUserPermissions(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, up := range list {
		seen[up.PermissionStr] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out, nil
}

// AddRoleInheritance добавляет наследование ролей childRole <- parentRole в пределах одного workspace.
// В Casbin это выражается через групповую политику g(\"role:child\", \"role:parent\", workspaceID).
func (s *Service) AddRoleInheritance(ctx context.Context, workspaceID, childRoleID, parentRoleID string) error {
	if childRoleID == parentRoleID {
		return errors.New("child and parent roles must be different")
	}
	child, err := s.repo.GetRoleByID(ctx, childRoleID)
	if err != nil || child == nil {
		return permRepo.ErrRoleNotFound
	}
	parent, err := s.repo.GetRoleByID(ctx, parentRoleID)
	if err != nil || parent == nil {
		return permRepo.ErrRoleNotFound
	}
	if child.WorkspaceID != workspaceID || parent.WorkspaceID != workspaceID {
		return errors.New("roles must belong to the same workspace")
	}
	if err := s.repo.CreateRoleInheritance(ctx, workspaceID, childRoleID, parentRoleID); err != nil {
		return err
	}
	_, err = s.enforcer.AddGroupingPolicy("role:"+child.Name, "role:"+parent.Name, workspaceID)
	if err != nil {
		_ = s.repo.DeleteRoleInheritance(ctx, workspaceID, childRoleID, parentRoleID)
		return err
	}
	return s.enforcer.SavePolicy()
}

// RemoveRoleInheritance удаляет наследование ролей childRole <- parentRole.
func (s *Service) RemoveRoleInheritance(ctx context.Context, workspaceID, childRoleID, parentRoleID string) error {
	child, err := s.repo.GetRoleByID(ctx, childRoleID)
	if err != nil || child == nil {
		return permRepo.ErrRoleNotFound
	}
	parent, err := s.repo.GetRoleByID(ctx, parentRoleID)
	if err != nil || parent == nil {
		return permRepo.ErrRoleNotFound
	}
	if err := s.repo.DeleteRoleInheritance(ctx, workspaceID, childRoleID, parentRoleID); err != nil {
		return err
	}
	_, _ = s.enforcer.RemoveGroupingPolicy("role:"+child.Name, "role:"+parent.Name, workspaceID)
	return s.enforcer.SavePolicy()
}

// parsePermission разбивает "module:entity:action" на obj="module:entity" и act="action".
func parsePermission(perm string) (obj, act string, err error) {
	parts := strings.SplitN(perm, ":", 3)
	if len(parts) != 3 {
		return "", "", ErrInvalidPermForm
	}
	return parts[0] + ":" + parts[1], parts[2], nil
}

// EnsureSystemRolePolicies загружает базовые политики для системных ролей во все workspace (вызов при старте или миграции).
func (s *Service) EnsureSystemRolePolicies(ctx context.Context) error {
	return s.ensureSystemPoliciesForAllWorkspaces(ctx)
}

// SyncGroupingPoliciesFromAssignments загружает все user_role_assignments в Casbin как g(user:id, role:name, workspace_id).
// Сначала удаляет все групповые политики, затем добавляет из БД (идемпотентно при повторном вызове).
func (s *Service) SyncGroupingPoliciesFromAssignments(ctx context.Context) error {
	existing, _ := s.enforcer.GetGroupingPolicy()
	for _, p := range existing {
		if len(p) >= 3 {
			_, _ = s.enforcer.RemoveGroupingPolicy(p[0], p[1], p[2])
		}
	}
	// Пользовательские назначения ролей
	assignments, err := s.repo.ListAllUserRoleAssignments(ctx)
	if err != nil {
		return err
	}
	for _, a := range assignments {
		role, _ := s.repo.GetRoleByID(ctx, a.RoleID)
		if role != nil {
			_, _ = s.enforcer.AddGroupingPolicy("user:"+a.UserID, "role:"+role.Name, a.WorkspaceID)
		}
	}
	// Наследование ролей: g(\"role:child\", \"role:parent\", workspace)
	workspaceIDs, err := s.repo.ListDistinctWorkspaceIDs(ctx)
	if err != nil {
		return err
	}
	for _, wid := range workspaceIDs {
		inheritance, err := s.repo.ListRoleInheritanceByWorkspace(ctx, wid)
		if err != nil {
			return err
		}
		for _, ri := range inheritance {
			child, _ := s.repo.GetRoleByID(ctx, ri.ChildRoleID)
			parent, _ := s.repo.GetRoleByID(ctx, ri.ParentRoleID)
			if child != nil && parent != nil {
				_, _ = s.enforcer.AddGroupingPolicy("role:"+child.Name, "role:"+parent.Name, wid)
			}
		}
	}
	return s.enforcer.SavePolicy()
}

func (s *Service) ensureSystemPoliciesForAllWorkspaces(ctx context.Context) error {
	// Получить список workspace_id из workspace_roles (distinct)
	// Для этого добавим в repository метод или используем существующий workspace repo. Проще добавить в permission repo.
	workspaceIDs, err := s.repo.ListDistinctWorkspaceIDs(ctx)
	if err != nil {
		return err
	}
	for _, wid := range workspaceIDs {
		s.addSystemPoliciesForWorkspace(wid)
	}
	return s.enforcer.SavePolicy()
}

// SeedSystemPoliciesForWorkspace добавляет базовые политики для OWNER/ADMIN/MEMBER/GUEST в один workspace.
func (s *Service) SeedSystemPoliciesForWorkspace(workspaceID string) {
	s.addSystemPoliciesForWorkspace(workspaceID)
}

// SavePolicy сохраняет политики Casbin (адаптер/файл).
func (s *Service) SavePolicy() error {
	return s.enforcer.SavePolicy()
}

func (s *Service) addSystemPoliciesForWorkspace(workspaceID string) {
	systemRoles := []string{"role:OWNER", "role:ADMIN", "role:MEMBER", "role:GUEST"}
	for _, sub := range systemRoles {
		policies, _ := s.enforcer.GetFilteredPolicy(0, sub)
		for _, p := range policies {
			if len(p) >= 4 && p[1] == workspaceID {
				_, _ = s.enforcer.RemovePolicy(p[0], p[1], p[2], p[3])
			}
		}
	}
	for _, p := range ownerAdminPolicies {
		_, _ = s.enforcer.AddPolicy("role:OWNER", workspaceID, p.obj, p.act)
		_, _ = s.enforcer.AddPolicy("role:ADMIN", workspaceID, p.obj, p.act)
	}
	for _, p := range memberBasePolicies {
		_, _ = s.enforcer.AddPolicy("role:MEMBER", workspaceID, p.obj, p.act)
	}
	for _, p := range guestBasePolicies {
		_, _ = s.enforcer.AddPolicy("role:GUEST", workspaceID, p.obj, p.act)
	}
}

type objAct struct{ obj, act string }

var (
	ownerAdminPolicies = []objAct{
		{"crm:deal", "create"}, {"crm:deal", "read"}, {"crm:deal", "update"}, {"crm:deal", "delete"}, {"crm:deal", "move"},
		{"crm:contact", "create"}, {"crm:contact", "read"}, {"crm:contact", "update"}, {"crm:contact", "delete"},
		{"crm:company", "create"}, {"crm:company", "read"}, {"crm:company", "update"}, {"crm:company", "delete"},
		{"crm:pipeline", "manage"}, {"crm:activity", "create"}, {"crm:activity", "read"}, {"crm:activity", "update"}, {"crm:activity", "delete"},
		{"crm:export", "deals"},
		{"habits:habit", "create"}, {"habits:habit", "read"}, {"habits:habit", "update"}, {"habits:habit", "delete"}, {"habits:habit", "complete"},
		{"habits:journal", "create"}, {"habits:journal", "read"}, {"habits:journal", "update"}, {"habits:journal", "delete"},
		{"projects:project", "create"}, {"projects:project", "read"}, {"projects:project", "update"}, {"projects:project", "delete"},
		{"projects:entity", "attach"}, {"projects:entity", "detach"},
		{"workspace:member", "invite"}, {"workspace:member", "remove"}, {"workspace:role", "manage"},
		{"workspace:module", "read"}, {"workspace:module", "manage"},
	}
	memberBasePolicies = []objAct{
		{"crm:deal", "create"}, {"crm:deal", "read"}, {"crm:deal", "update"}, {"crm:deal", "move"},
		{"crm:contact", "create"}, {"crm:contact", "read"}, {"crm:contact", "update"},
		{"crm:company", "create"}, {"crm:company", "read"}, {"crm:company", "update"},
		{"habits:habit", "create"}, {"habits:habit", "read"}, {"habits:habit", "update"}, {"habits:habit", "complete"},
		{"habits:journal", "create"}, {"habits:journal", "read"}, {"habits:journal", "update"},
		{"projects:project", "create"}, {"projects:project", "read"}, {"projects:project", "update"},
		{"workspace:member", "invite"},
		{"workspace:module", "read"}, {"workspace:module", "manage"},
	}
	guestBasePolicies = []objAct{
		{"crm:deal", "read"}, {"crm:contact", "read"}, {"crm:company", "read"},
		{"habits:habit", "read"}, {"habits:journal", "read"},
		{"projects:project", "read"},
		{"workspace:module", "read"}, // просмотр списка модулей (GET /modules)
	}
)
