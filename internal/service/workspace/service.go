package workspace

import (
	"context"
	"errors"
	"log"

	"backend/internal/model"
	permService "backend/internal/service/permission"
	"backend/internal/repository/license"
	"backend/internal/repository/user_preferences"
	"backend/internal/repository/workspace"

	"github.com/google/uuid"
)

var (
	ErrWorkspaceNotFound   = errors.New("workspace not found")
	ErrAccessDenied        = errors.New("access denied")
	ErrNoActiveWorkspace   = errors.New("no active workspace")
	ErrLicenseRequired     = errors.New("license required: purchase module or request admin grant")
	ErrMemberNotFound      = errors.New("member not found")
	ErrCannotRemoveSelf    = errors.New("cannot remove yourself from workspace")
	ErrCannotRemoveOwner   = errors.New("cannot remove the sole owner")
	ErrCannotChangeOwnRole = errors.New("cannot change your own role")
	ErrCannotChangeOwner   = errors.New("cannot change role of the sole owner")
)

type Service struct {
	repo        *workspace.Repository
	prefRepo    *user_preferences.Repository
	licenseRepo *license.Repository
	permSvc     *permService.Service
}

func NewService(repo *workspace.Repository, prefRepo *user_preferences.Repository, licenseRepo *license.Repository, permSvc *permService.Service) *Service {
	return &Service{
		repo:        repo,
		prefRepo:    prefRepo,
		licenseRepo: licenseRepo,
		permSvc:     permSvc,
	}
}

// List возвращает только воркспейсы, в которых пользователь состоит (owner или user_workspaces). Дропдаун в сайдбаре — только свои.
// Админ может перейти в чужой воркспейс через админ-панель (Switch при этом разрешён), но в общем списке — только свои.
func (s *Service) List(ctx context.Context, userID string, userRole model.UserRole) ([]model.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, uid)
}

func (s *Service) Create(ctx context.Context, dto model.CreateWorkspaceDto, userID string) (*model.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	ws, err := s.repo.Create(ctx, dto, uid)
	if err != nil {
		return nil, err
	}

	// Триггер создал workspace_roles. Заливаем политики Casbin для нового workspace
	// (EnsureSystemRolePolicies при старте не видит workspace, созданные в runtime).
	s.permSvc.SeedSystemPoliciesForWorkspace(ws.ID)

	// Назначаем роль OWNER в user_role_assignments для Casbin (user_workspaces уже добавлен в repo).
	if err := s.permSvc.AssignRoleByName(ctx, userID, ws.ID, "OWNER", userID); err != nil {
		log.Printf("Workspace Create: failed to assign OWNER role to creator %s in workspace %s: %v", userID, ws.ID, err)
	}

	return ws, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, userID string, userRole model.UserRole) (*model.Workspace, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	hasAccess, err := s.repo.CheckAccess(ctx, wsID, uid, userRole)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, ErrAccessDenied
	}

	ws, err := s.repo.Get(ctx, wsID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, ErrWorkspaceNotFound
	}

	return ws, nil
}

func (s *Service) Update(ctx context.Context, workspaceID string, dto model.UpdateWorkspaceDto, userID string, userRole model.UserRole) (*model.Workspace, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	hasAccess, err := s.repo.CheckAccess(ctx, wsID, uid, userRole)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, ErrAccessDenied
	}

	ws, err := s.repo.Update(ctx, wsID, dto)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, ErrWorkspaceNotFound
	}

	return ws, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, userID string, userRole model.UserRole) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	hasAccess, err := s.repo.CheckAccess(ctx, wsID, uid, userRole)
	if err != nil {
		return err
	}
	if !hasAccess {
		return ErrAccessDenied
	}

	return s.repo.Delete(ctx, wsID)
}

func (s *Service) SetCurrentWorkspace(ctx context.Context, userID, workspaceID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return s.prefRepo.SetCurrentWorkspace(ctx, uid, workspaceID)
}

func (s *Service) UnsetCurrentWorkspace(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return s.prefRepo.UnsetCurrentWorkspace(ctx, uid)
}

// GetCurrentWorkspace возвращает текущий воркспейс пользователя. Для ADMIN при отсутствии выбора — первый из всех воркспейсов.
func (s *Service) GetCurrentWorkspace(ctx context.Context, userID string, userRole model.UserRole) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	workspaceID, err := s.prefRepo.GetCurrentWorkspace(ctx, uid)
	if err != nil {
		return "", err
	}
	if workspaceID != "" {
		return workspaceID, nil
	}
	var list []model.Workspace
	if userRole == model.UserRoleAdmin {
		list, err = s.repo.ListAll(ctx)
	} else {
		list, err = s.repo.List(ctx, uid)
	}
	if err != nil || len(list) == 0 {
		return "", ErrNoActiveWorkspace
	}
	workspaceID = list[0].ID
	_ = s.prefRepo.SetCurrentWorkspace(ctx, uid, workspaceID)
	return workspaceID, nil
}

// HasAccess проверяет доступ к воркспейсу. Глобальный ADMIN имеет доступ ко всем воркспейсам.
func (s *Service) HasAccess(ctx context.Context, workspaceID, userID string, userRole model.UserRole) (bool, error) {
	if userRole == model.UserRoleAdmin {
		return true, nil
	}
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return false, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	return s.repo.HasAccess(ctx, wsID, uid)
}

func (s *Service) ListAllForAdmin(ctx context.Context) ([]model.Workspace, error) {
	return s.repo.ListAll(ctx)
}

// GetWorkspaceModules возвращает модули воркспейса с реальным статусом (active/disabled) из workspace_modules.
// Для админа и обычного пользователя — один и тот же источник: фактические записи в workspace_modules для этого workspace.
func (s *Service) GetWorkspaceModules(ctx context.Context, workspaceID, userID string, userRole model.UserRole) ([]model.WorkspaceModuleInfo, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	// Админ имеет доступ к любому воркспейсу; проверку доступа не делаем. Читаем реальное состояние модулей.
	if userRole != model.UserRoleAdmin {
		uid, err := uuid.Parse(userID)
		if err != nil {
			return nil, err
		}
		hasAccess, err := s.repo.HasAccess(ctx, wsID, uid)
		if err != nil || !hasAccess {
			return nil, ErrAccessDenied
		}
	}
	return s.repo.ListAllModulesWithWorkspaceState(ctx, wsID)
}

var ErrModuleNotFound = errors.New("module not found")

// EnableModule включает модуль в workspace. Разрешено только владельцу воркспейса или глобальному админу.
// Для не-core модуля у владельца воркспейса должна быть активная лицензия (все воркспейсы или этот).
func (s *Service) EnableModule(ctx context.Context, workspaceID, userID string, userRole model.UserRole, moduleCode string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	if userRole != model.UserRoleAdmin {
		ok, err := s.repo.IsOwner(ctx, wsID, uid)
		if err != nil || !ok {
			return ErrAccessDenied
		}
	}

	mod, err := s.repo.GetModuleByCode(ctx, moduleCode)
	if err != nil {
		return err
	}
	if mod == nil {
		return ErrModuleNotFound
	}

	modID, err := uuid.Parse(mod.ID)
	if err != nil {
		return err
	}

	// Админ может включать без лицензии. Core-модули — без лицензии. Остальное — только при наличии лицензии у владельца.
	if userRole != model.UserRoleAdmin && !mod.IsCore {
		has, err := s.licenseRepo.HasLicense(ctx, uid, modID, &wsID)
		if err != nil {
			return err
		}
		if !has {
			return ErrLicenseRequired
		}
	}

	return s.repo.AddWorkspaceModule(ctx, wsID, modID)
}

// DisableModule отключает модуль в workspace. Разрешено только владельцу воркспейса или глобальному админу.
func (s *Service) DisableModule(ctx context.Context, workspaceID, userID string, userRole model.UserRole, moduleCode string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	if userRole != model.UserRoleAdmin {
		ok, err := s.repo.IsOwner(ctx, wsID, uid)
		if err != nil || !ok {
			return ErrAccessDenied
		}
	}

	mod, err := s.repo.GetModuleByCode(ctx, moduleCode)
	if err != nil {
		return err
	}
	if mod == nil {
		return ErrModuleNotFound
	}

	modID, err := uuid.Parse(mod.ID)
	if err != nil {
		return err
	}
	return s.repo.SetWorkspaceModuleStatus(ctx, wsID, modID, model.WorkspaceModuleStatusDisabled)
}

// ListMyLicenses возвращает активные лицензии текущего пользователя (для UI: какие модули можно включать).
func (s *Service) ListMyLicenses(ctx context.Context, userID string) ([]model.UserModuleLicense, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.licenseRepo.ListByUserID(ctx, uid)
}

// CanEnableModuleInWorkspace проверяет, может ли пользователь включить модуль в данном воркспейсе
// (владелец или админ + лицензия для не-core, или core).
func (s *Service) CanEnableModuleInWorkspace(ctx context.Context, workspaceID, userID string, userRole model.UserRole, moduleCode string) (bool, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return false, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	if userRole == model.UserRoleAdmin {
		return true, nil
	}
	ok, err := s.repo.IsOwner(ctx, wsID, uid)
	if err != nil || !ok {
		return false, nil
	}
	mod, err := s.repo.GetModuleByCode(ctx, moduleCode)
	if err != nil || mod == nil {
		return false, nil
	}
	if mod.IsCore {
		return true, nil
	}
	modID, _ := uuid.Parse(mod.ID)
	return s.licenseRepo.HasLicense(ctx, uid, modID, &wsID)
}

// ListMembers возвращает участников workspace. Доступ: любой участник workspace.
func (s *Service) ListMembers(ctx context.Context, workspaceID, userID string, userRole model.UserRole) ([]model.WorkspaceMember, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	hasAccess, err := s.repo.HasAccess(ctx, wsID, uid)
	if err != nil {
		return nil, err
	}
	if !hasAccess && userRole != model.UserRoleAdmin {
		return nil, ErrAccessDenied
	}
	return s.repo.ListMembers(ctx, wsID)
}

// RemoveMember удаляет участника из workspace. Только OWNER/ADMIN.
func (s *Service) RemoveMember(ctx context.Context, workspaceID, userID, targetUserID string, userRole model.UserRole) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	targetUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return err
	}
	if userID == targetUserID {
		return ErrCannotRemoveSelf
	}
	if userRole != model.UserRoleAdmin {
		callerRole, err := s.repo.GetUserWorkspaceRole(ctx, wsID, uid)
		if err != nil || (callerRole != "OWNER" && callerRole != "ADMIN") {
			return ErrAccessDenied
		}
	}
	ownerCount, err := s.repo.CountOwners(ctx, wsID)
	if err != nil {
		return err
	}
	targetRole, err := s.repo.GetUserWorkspaceRole(ctx, wsID, targetUID)
	if err != nil {
		return err
	}
	if targetRole == "" {
		return ErrMemberNotFound
	}
	if targetRole == "OWNER" && ownerCount <= 1 {
		return ErrCannotRemoveOwner
	}
	// Сначала снимаем роли через permission service (Casbin + user_role_assignments)
	rolesFull, err := s.permSvc.GetUserRolesFull(ctx, targetUserID, workspaceID)
	if err != nil {
		return err
	}
	for _, role := range rolesFull {
		_ = s.permSvc.RemoveRole(ctx, targetUserID, role.ID, workspaceID)
	}
	// Затем удаляем из user_workspaces
	if err := s.repo.RemoveMember(ctx, wsID, targetUID); err != nil {
		return err
	}
	// Сбрасываем current_workspace_id у удалённого пользователя, если он указывал на этот workspace
	if cur, _ := s.prefRepo.GetCurrentWorkspace(ctx, targetUID); cur == workspaceID {
		_ = s.prefRepo.UnsetCurrentWorkspace(ctx, targetUID)
	}
	return nil
}

// UpdateMemberRole меняет системную роль участника. Только OWNER/ADMIN.
func (s *Service) UpdateMemberRole(ctx context.Context, workspaceID, userID, targetUserID, newRole string, userRole model.UserRole) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	targetUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return err
	}
	if userRole != model.UserRoleAdmin {
		callerRole, err := s.repo.GetUserWorkspaceRole(ctx, wsID, uid)
		if err != nil || (callerRole != "OWNER" && callerRole != "ADMIN") {
			return ErrAccessDenied
		}
	}
	if userID == targetUserID {
		return ErrCannotChangeOwnRole
	}
	targetRole, err := s.repo.GetUserWorkspaceRole(ctx, wsID, targetUID)
	if err != nil || targetRole == "" {
		return ErrMemberNotFound
	}
	ownerCount, err := s.repo.CountOwners(ctx, wsID)
	if err != nil {
		return err
	}
	if targetRole == "OWNER" && ownerCount <= 1 {
		return ErrCannotChangeOwner
	}
	// Обновить user_workspaces.role
	if err := s.repo.UpdateUserWorkspaceRole(ctx, wsID, targetUID, newRole); err != nil {
		return err
	}
	// Снять старую системную роль и назначить новую (user_role_assignments + Casbin)
	rolesFull, err := s.permSvc.GetUserRolesFull(ctx, targetUserID, workspaceID)
	if err != nil {
		return err
	}
	for _, role := range rolesFull {
		if role.IsSystem {
			_ = s.permSvc.RemoveRole(ctx, targetUserID, role.ID, workspaceID)
		}
	}
	return s.permSvc.AssignRoleByName(ctx, targetUserID, workspaceID, newRole, userID)
}

// GrantLicense выдаёт лицензию пользователю (только админ). Для теста или промо до момента оплаты.
func (s *Service) GrantLicense(ctx context.Context, targetUserID, moduleCode, scope string, workspaceID *string) (*model.UserModuleLicense, error) {
	uid, err := uuid.Parse(targetUserID)
	if err != nil {
		return nil, err
	}
	mod, err := s.repo.GetModuleByCode(ctx, moduleCode)
	if err != nil || mod == nil {
		return nil, ErrModuleNotFound
	}
	modID, _ := uuid.Parse(mod.ID)
	if scope != model.LicenseScopeAllWorkspaces && scope != model.LicenseScopeSingleWorkspace {
		return nil, errors.New("scope must be all_workspaces or single_workspace")
	}
	if scope == model.LicenseScopeSingleWorkspace && (workspaceID == nil || *workspaceID == "") {
		return nil, errors.New("workspace_id required for single_workspace")
	}
	lic := &model.UserModuleLicense{
		UserID:   uid.String(),
		ModuleID: modID.String(),
		Scope:    scope,
		WorkspaceID: workspaceID,
		Status:   model.LicenseStatusActive,
		Source:   model.LicenseSourceAdminGrant,
	}
	if err := s.licenseRepo.Create(ctx, lic); err != nil {
		return nil, err
	}
	lic.ModuleCode = mod.Code
	return lic, nil
}
