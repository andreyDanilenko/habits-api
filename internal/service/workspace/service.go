package workspace

import (
	"context"
	"errors"

	"backend/internal/model"
	"backend/internal/repository/license"
	"backend/internal/repository/user_preferences"
	"backend/internal/repository/workspace"

	"github.com/google/uuid"
)

var (
	ErrWorkspaceNotFound  = errors.New("workspace not found")
	ErrAccessDenied       = errors.New("access denied")
	ErrNoActiveWorkspace  = errors.New("no active workspace")
	ErrLicenseRequired    = errors.New("license required: purchase module or request admin grant")
)

type Service struct {
	repo       *workspace.Repository
	prefRepo   *user_preferences.Repository
	licenseRepo *license.Repository
}

func NewService(repo *workspace.Repository, prefRepo *user_preferences.Repository, licenseRepo *license.Repository) *Service {
	return &Service{
		repo:        repo,
		prefRepo:    prefRepo,
		licenseRepo: licenseRepo,
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

	return s.repo.Create(ctx, dto, uid)
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
