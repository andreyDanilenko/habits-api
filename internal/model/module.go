package model

// Module — справочник модулей системы (habits, crm, ...).
// Используется для проверки, какие модули вообще существуют и какой из них core.
type Module struct {
	ID          string `json:"id" db:"id"`
	Code        string `json:"code" db:"code"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	IsCore      bool   `json:"isCore" db:"is_core"`
	CreatedAt   string `json:"createdAt" db:"created_at"`
}

// WorkspaceModuleStatus — статус модуля в workspace (оплачен/триал/выключен).
const (
	WorkspaceModuleStatusActive   = "active"
	WorkspaceModuleStatusTrial    = "trial"
	WorkspaceModuleStatusDisabled = "disabled"
)

// WorkspaceModule — привязка модуля к workspace: модуль включён/оплачен в этом workspace.
// Доступ к сущностям модуля (например habits) проверять так:
// пользователь в user_workspaces для workspace_id И есть запись здесь со status = active.
type WorkspaceModule struct {
	ID          string  `json:"id" db:"id"`
	WorkspaceID string  `json:"workspaceId" db:"workspace_id"`
	ModuleID    string  `json:"moduleId" db:"module_id"`
	Status      string  `json:"status" db:"status"` // active, trial, disabled
	ActivatedAt string  `json:"activatedAt" db:"activated_at"`
	ExpiresAt   *string `json:"expiresAt,omitempty" db:"expires_at"`
	Settings    []byte  `json:"settings,omitempty" db:"settings"` // JSONB
	CreatedAt   string  `json:"createdAt" db:"created_at"`
}

// Коды модулей (совпадают с modules.code в БД).
const (
	ModuleCodeHabits = "habits"
	ModuleCodeCRM   = "crm"
)

// WorkspaceModuleInfo — ответ API для списка модулей workspace (для фронта).
type WorkspaceModuleInfo struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	ModuleID    string `json:"moduleId"`
	ModuleName  string `json:"moduleName"` // code из modules (habits, crm)
	Status      string `json:"status"`
	Enabled     bool   `json:"enabled"` // true если status == active
	Settings    any    `json:"config,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// Scope лицензии на модуль.
const (
	LicenseScopeAllWorkspaces   = "all_workspaces"
	LicenseScopeSingleWorkspace = "single_workspace"
)

// UserModuleLicense — лицензия пользователя на модуль (куплен для всех воркспейсов или для одного).
type UserModuleLicense struct {
	ID          string  `json:"id" db:"id"`
	UserID      string  `json:"userId" db:"user_id"`
	ModuleID    string  `json:"moduleId" db:"module_id"`
	ModuleCode  string  `json:"moduleCode" db:"-"` // заполняется при выдаче API
	Scope       string  `json:"scope" db:"scope"`
	WorkspaceID *string `json:"workspaceId,omitempty" db:"workspace_id"`
	Status      string  `json:"status" db:"status"`
	Source      string  `json:"source" db:"source"`
	ExpiresAt   *string `json:"expiresAt,omitempty" db:"expires_at"`
	CreatedAt   string  `json:"createdAt" db:"created_at"`
	UpdatedAt   string  `json:"updatedAt" db:"updated_at"`
}

const (
	LicenseStatusActive    = "active"
	LicenseStatusExpired   = "expired"
	LicenseStatusCancelled = "cancelled"
	LicenseSourcePurchase  = "purchase"
	LicenseSourceAdminGrant = "admin_grant"
)
