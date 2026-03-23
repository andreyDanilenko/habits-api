package permission

import (
	"context"
	"errors"
	"strings"

	"backend/internal/model"
	permRepo "backend/internal/repository/permission"

	"github.com/google/uuid"
)

// DataScope — видимость строк для object_key (crm:deal и т.д.).
type DataScope string

const (
	DataScopeAll        DataScope = "all"
	DataScopeOwner      DataScope = "owner"
	DataScopeDepartment DataScope = "department"
	DataScopeNone       DataScope = "none"
)

var errInvalidDataScope = errors.New("dataScope must be one of: all, owner, department, none")

func normalizeDataScope(s string) (DataScope, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(DataScopeAll):
		return DataScopeAll, nil
	case string(DataScopeOwner):
		return DataScopeOwner, nil
	case string(DataScopeDepartment):
		return DataScopeDepartment, nil
	case string(DataScopeNone):
		return DataScopeNone, nil
	default:
		return "", errInvalidDataScope
	}
}

func dataScopeRank(s DataScope) int {
	switch s {
	case DataScopeAll:
		return 4
	case DataScopeDepartment:
		return 3
	case DataScopeOwner:
		return 2
	case DataScopeNone:
		return 1
	default:
		return 4
	}
}

func rankToDataScope(r int) DataScope {
	switch r {
	case 4:
		return DataScopeAll
	case 3:
		return DataScopeDepartment
	case 2:
		return DataScopeOwner
	case 1:
		return DataScopeNone
	default:
		return DataScopeAll
	}
}

// GetEffectiveDataScope агрегирует scope по ролям пользователя в workspace.
// Та же логика «override» кастомных ролей, что у GetEffectivePermissions: при наличии кастомной роли
// системные роли не учитываются. Между учитываемыми ролями берётся наиболее широкий scope (all > department > owner > none).
// При отсутствии строк в role_object_scopes для роли используется all (как раньше — без ограничения списка).
//
// Владелец workspace (workspaces.owner_id) всегда получает all — нельзя «запереть» себя ограничениями ролей.
func (s *Service) GetEffectiveDataScope(ctx context.Context, userID, workspaceID, objectKey string) (DataScope, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return DataScopeAll, nil
	}
	wsUUID, errWS := uuid.Parse(workspaceID)
	userUUID, errU := uuid.Parse(userID)
	if errWS == nil && errU == nil && s.workspaceRepo != nil {
		isOwner, err := s.workspaceRepo.IsOwner(ctx, wsUUID, userUUID)
		if err == nil && isOwner {
			return DataScopeAll, nil
		}
	}
	rolesFull, err := s.GetUserRolesFull(ctx, userID, workspaceID)
	if err != nil {
		return DataScopeAll, err
	}
	hasCustomRole := false
	for _, r := range rolesFull {
		if !r.IsSystem {
			hasCustomRole = true
			break
		}
	}
	maxRank := 0
	any := false
	for _, role := range rolesFull {
		if hasCustomRole && role.IsSystem {
			continue
		}
		any = true
		list, err := s.repo.ListRoleObjectScopes(ctx, role.ID)
		if err != nil {
			return DataScopeAll, err
		}
		var scopeStr string
		for _, row := range list {
			if row.ObjectKey == objectKey {
				scopeStr = row.DataScope
				break
			}
		}
		ds := DataScopeAll
		if scopeStr != "" {
			norm, err := normalizeDataScope(scopeStr)
			if err != nil {
				return DataScopeAll, err
			}
			ds = norm
		}
		if dr := dataScopeRank(ds); dr > maxRank {
			maxRank = dr
		}
	}
	if !any {
		return DataScopeAll, nil
	}
	return rankToDataScope(maxRank), nil
}

// ListRoleObjectScopes для UI редактирования роли.
func (s *Service) ListRoleObjectScopes(ctx context.Context, workspaceID, roleID string) ([]model.RoleObjectScope, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, permRepo.ErrRoleNotFound
	}
	if role.WorkspaceID != workspaceID {
		return nil, permRepo.ErrRoleNotFound
	}
	return s.repo.ListRoleObjectScopes(ctx, roleID)
}

// SetRoleObjectScopes полностью задаёт map object_key → data_scope для роли.
func (s *Service) SetRoleObjectScopes(ctx context.Context, workspaceID, roleID string, scopes map[string]string) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return permRepo.ErrRoleNotFound
	}
	if role.WorkspaceID != workspaceID {
		return errors.New("role does not belong to this workspace")
	}
	if role.IsSystem && strings.EqualFold(role.Name, "OWNER") {
		return ErrProtectedRoleObjectScopes
	}
	normalized := make(map[string]string, len(scopes))
	for k, v := range scopes {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, err := normalizeDataScope(v); err != nil {
			return err
		}
		normalized[k] = strings.ToLower(strings.TrimSpace(v))
	}
	return s.repo.ReplaceRoleObjectScopes(ctx, roleID, normalized)
}

// Ключи object scope, для которых отдаём значения в /me/permissions (фронт: уведомления, UI).
var meDataScopeKeys = []string{"crm:deal", "crm:contact", "crm:company"}

// MeDataScopeContext — карта dataScopes + опционально department_id текущего пользователя (для фильтра уведомлений).
func (s *Service) MeDataScopeContext(ctx context.Context, userID, workspaceID string, globalAdmin bool) (dataScopes map[string]string, departmentID string, departmentOK bool, err error) {
	dataScopes = make(map[string]string, len(meDataScopeKeys))
	if globalAdmin {
		for _, k := range meDataScopeKeys {
			dataScopes[k] = string(DataScopeAll)
		}
	} else {
		for _, k := range meDataScopeKeys {
			ds, e := s.GetEffectiveDataScope(ctx, userID, workspaceID, k)
			if e != nil {
				return nil, "", false, e
			}
			dataScopes[k] = string(ds)
		}
	}
	if s.userDept != nil {
		if d, ok, e := s.userDept.GetDepartmentID(ctx, userID); e == nil && ok {
			return dataScopes, d, true, nil
		}
	}
	return dataScopes, "", false, nil
}
