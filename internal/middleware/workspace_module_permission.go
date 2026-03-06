// Package middleware: WorkspacePathMiddleware, ModuleLicenseMiddleware, PermissionMiddleware.
//
// Дизайн рассчитан на полную гибкость:
//   - Админка воркспейса: участники (invite/remove), модули (manage), роли (manage) — права из permission_catalog.
//   - Модули (CRM, Habits, Projects): гранулярные действия create/read/update/delete по сущностям.
//   - Системная админка (/admin/*): пока на глобальной роли (RequireAdmin); при необходимости можно ввести system:* в каталог и маппинг.
//   - После включения боевого режима PermissionMiddleware все решения будут приниматься по Casbin (роли + индивидуальные права).
package middleware

import (
	"log"
	"net/http"
	"strings"

	"backend/internal/model"
	permissionService "backend/internal/service/permission"
	"backend/internal/service/workspace"
	"backend/pkg/response"

	casbin "github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

// GinNoWorkspaceModuleReadKey — при отсутствии workspace:module:read для GET /modules
// middleware не возвращает 403, а устанавливает этот флаг; хендлер отвечает { modules: [] }.
const GinNoWorkspaceModuleReadKey = "no_workspace_module_read"

// WorkspaceMiddleware извлекает workspaceId из URL, проверяет членство пользователя и кладёт workspace_id в контекст Gin.
func WorkspacePathMiddleware(workspaceService *workspace.Service, responder *response.Responder) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Привязываемся к шаблону маршрута Gin, а не к фактическому URL.
		// Это позволяет отличить /workspaces/:workspaceId/... от /workspaces/current и /workspaces.
		pattern := c.FullPath()
		if pattern == "" {
			pattern = c.Request.URL.Path
		}

		// Если в шаблоне нет сегмента :workspaceId — этот эндпоинт не работает в контексте конкретного воркспейса.
		if !strings.Contains(pattern, "/workspaces/:workspaceId") {
			c.Next()
			return
		}

		workspaceID := c.Param("workspaceId")
		if workspaceID == "" {
			// Не смогли извлечь workspaceId для эндпоинта, который его требует — считаем, что это ошибка запроса.
			responder.BadRequest(c, "workspaceId is required in path")
			c.Abort()
			return
		}

		userID, userRole, userExists := GetAuthFromGin(c)

		// Если пользователя нет — доступ запрещён для защищённых workspace‑эндпоинтов
		if !userExists {
			responder.Unauthorized(c, "Authentication required for workspace access")
			c.Abort()
			return
		}

		// Проверяем доступ к workspace через WorkspaceService (учитывает глобальную роль ADMIN)
		ok, err := workspaceService.HasAccess(c.Request.Context(), workspaceID, userID, userRole)
		if err != nil {
			log.Printf("WorkspacePathMiddleware: error checking access: %v", err)
			responder.InternalServerError(c, "Failed to check workspace access")
			c.Abort()
			return
		}
		if !ok {
			responder.Forbidden(c, "No access to this workspace")
			c.Abort()
			return
		}

		// Кладём workspace_id в Gin‑контекст для хендлеров и следующего middleware
		c.Set(GinWorkspaceIDKey, workspaceID)

		c.Next()
	}
}

// ModuleLicenseMiddleware проверяет, включен ли модуль в workspace и есть ли у пользователя лицензия (для не-core модулей).
func ModuleLicenseMiddleware(workspaceService *workspace.Service, responder *response.Responder) gin.HandlerFunc {
	return func(c *gin.Context) {
		wsVal, exists := c.Get(GinWorkspaceIDKey)
		if !exists {
			// Нет workspace в контексте — пропускаем (например, /auth/*)
			c.Next()
			return
		}
		workspaceID, _ := wsVal.(string)

		userID, userRole, userExists := GetAuthFromGin(c)

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		moduleCode := detectModuleCode(path)
		if moduleCode == "" {
			// Не модульный эндпоинт — пропускаем
			c.Next()
			return
		}

		// Проверяем, что модуль вообще существует и его статус в workspace через WorkspaceService
		ctx := c.Request.Context()
		modules, err := workspaceService.GetWorkspaceModules(ctx, workspaceID, userID, userRole)
		if err != nil {
			log.Printf("ModuleLicenseMiddleware: error getting workspace modules: %v", err)
			responder.InternalServerError(c, "Failed to check module access")
			c.Abort()
			return
		}

		var found *model.WorkspaceModuleInfo
		for i := range modules {
			if modules[i].ModuleName == moduleCode {
				found = &modules[i]
				break
			}
		}
		if found == nil || !found.Enabled {
			responder.Forbidden(c, "Module not enabled in this workspace")
			c.Abort()
			return
		}

		// Дополнительная проверка лицензии для не-core модулей:
		// используем существующую бизнес-логику CanEnableModuleInWorkspace как универсальный gate.
		if userExists && !found.IsCore {
			can, err := workspaceService.CanEnableModuleInWorkspace(ctx, workspaceID, userID, userRole, moduleCode)
			if err != nil {
				log.Printf("ModuleLicenseMiddleware: error checking license: %v", err)
				responder.InternalServerError(c, "Failed to check module license")
				c.Abort()
				return
			}
			if !can {
				responder.Forbidden(c, "No license for this module")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// PermissionMiddleware — обёртка над Casbin:
// 1) вычисляет (obj, act) по endpoint'у,
// 2) проверяет доступ через Casbin (роли и наследование),
// 3) при отказе дополнительно учитывает индивидуальные права пользователя,
// 4) при итоговом отказе блокирует доступ (403).
func PermissionMiddleware(enforcer *casbin.Enforcer, permSvc *permissionService.Service, responder *response.Responder) gin.HandlerFunc {
	return func(c *gin.Context) {
		wsVal, exists := c.Get(GinWorkspaceIDKey)
		if !exists {
			// Нет workspace — считаем, что эндпоинт публичный / не требует прав
			c.Next()
			return
		}
		workspaceID, _ := wsVal.(string)

		userID, userRole, userExists := GetAuthFromGin(c)
		if !userExists {
			// Защищённые эндпоинты workspace всегда требуют авторизации
			responder.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}

		// Глобальный ADMIN имеет полный доступ в любом workspace — пропускаем без проверки Casbin.
		if userRole == model.UserRoleAdmin {
			c.Next()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		method := c.Request.Method
		obj, act := mapEndpointToPermission(method, path)
		if obj == "" || act == "" {
			// Эндпоинт не требует явного права — пропускаем
			c.Next()
			return
		}

		sub := "user:" + userID
		allowed, err := enforcer.Enforce(sub, workspaceID, obj, act)
		if err != nil {
			log.Printf("PermissionMiddleware: enforce error: %v (sub=%s, dom=%s, obj=%s, act=%s)", err, sub, workspaceID, obj, act)
			responder.InternalServerError(c, "Failed to check permissions")
			c.Abort()
			return
		}

		// Если по ролям доступ запрещён — проверяем индивидуальные права пользователя.
		if !allowed {
			ctx := c.Request.Context()
			list, err := permSvc.GetUserPermissions(ctx, userID, workspaceID)
			if err != nil {
				log.Printf("PermissionMiddleware: error loading user permissions: %v (user=%s, ws=%s)", err, userID, workspaceID)
			} else {
				target := obj + ":" + act
				for _, up := range list {
					if up.PermissionStr == target {
						allowed = true
						break
					}
				}
			}
		}

		if !allowed {
			// GET /modules: при отсутствии workspace:module:read возвращаем пустой массив, а не 403.
			// Хендлер проверяет GinNoWorkspaceModuleReadKey и отвечает { modules: [] }.
			if obj == "workspace:module" && act == "read" && method == http.MethodGet {
				c.Set(GinNoWorkspaceModuleReadKey, true)
				c.Next()
				return
			}
			responder.Forbidden(c, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// detectModuleCode определяет код модуля по URL-пути.
func detectModuleCode(path string) string {
	switch {
	case strings.Contains(path, "/crm/"):
		return model.ModuleCodeCRM
	case strings.Contains(path, "/habits/") || strings.Contains(path, "/journal/"):
		return model.ModuleCodeHabits
	case strings.Contains(path, "/projects/"):
		return model.ModuleCodeProjects
	default:
		return ""
	}
}

// endpointPermissionRule задаёт маппинг (path substring, method) → (obj, act) для permission_catalog.
// workspaceOnly: правило применяется только если путь содержит /workspaces/.
type endpointPermissionRule struct {
	pathSubstr   string
	workspaceOnly bool
	methods      []string
	obj, act     string
}

// endpointPermissionTable — таблица эндпоинт → право. Порядок важен: первое совпадение возвращается.
// Добавление нового эндпоинта: добавить строку в таблицу.
var endpointPermissionTable = []endpointPermissionRule{
	// Workspace-админка (участники, модули, роли)
	{"/members", true, []string{http.MethodGet, http.MethodPost}, "workspace:member", "invite"},
	{"/members", true, []string{http.MethodDelete}, "workspace:member", "remove"},
	{"/permissions/system-roles", true, []string{http.MethodGet}, "workspace:module", "read"},
	{"/modules", true, []string{http.MethodGet}, "workspace:module", "read"},   // GUEST может просматривать список модулей
	{"/modules", true, []string{http.MethodPost, http.MethodDelete}, "workspace:module", "manage"},
	{"/roles", true, []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}, "workspace:role", "manage"},
	// CRM
	{"/crm/deals", false, []string{http.MethodGet}, "crm:deal", "read"},
	{"/crm/deals", false, []string{http.MethodPost}, "crm:deal", "create"},
	{"/crm/deals", false, []string{http.MethodPut, http.MethodPatch}, "crm:deal", "update"},
	{"/crm/deals", false, []string{http.MethodDelete}, "crm:deal", "delete"},
	{"/crm/contacts", false, []string{http.MethodGet}, "crm:contact", "read"},
	{"/crm/contacts", false, []string{http.MethodPost}, "crm:contact", "create"},
	{"/crm/contacts", false, []string{http.MethodPut, http.MethodPatch}, "crm:contact", "update"},
	{"/crm/contacts", false, []string{http.MethodDelete}, "crm:contact", "delete"},
	{"/crm/companies", false, []string{http.MethodGet}, "crm:company", "read"},
	{"/crm/companies", false, []string{http.MethodPost}, "crm:company", "create"},
	{"/crm/companies", false, []string{http.MethodPut, http.MethodPatch}, "crm:company", "update"},
	{"/crm/companies", false, []string{http.MethodDelete}, "crm:company", "delete"},
	// Habits
	{"/habits/activities", false, []string{http.MethodGet}, "habits:habit", "read"},
	{"/habits/habits", false, []string{http.MethodGet}, "habits:habit", "read"},
	{"/habits/habits", false, []string{http.MethodPost}, "habits:habit", "create"},
	{"/habits/habits", false, []string{http.MethodPut, http.MethodPatch}, "habits:habit", "update"},
	{"/habits/habits", false, []string{http.MethodDelete}, "habits:habit", "delete"},
	{"/habits/journal", false, []string{http.MethodGet}, "habits:journal", "read"},
	{"/habits/journal", false, []string{http.MethodPost}, "habits:journal", "create"},
	{"/habits/journal", false, []string{http.MethodPut, http.MethodPatch}, "habits:journal", "update"},
	{"/habits/journal", false, []string{http.MethodDelete}, "habits:journal", "delete"},
	// Projects
	{"/projects", false, []string{http.MethodGet}, "projects:project", "read"},
	{"/projects", false, []string{http.MethodPost}, "projects:project", "create"},
	{"/projects", false, []string{http.MethodPut, http.MethodPatch}, "projects:project", "update"},
	{"/projects", false, []string{http.MethodDelete}, "projects:project", "delete"},
}

func methodIn(list []string, m string) bool {
	for _, x := range list {
		if x == m {
			return true
		}
	}
	return false
}

// mapEndpointToPermission маппит HTTP-метод и путь на (object, action) из permission_catalog.
// Системная админка (/admin/*) остаётся на RequireAdmin, без прав из каталога.
func mapEndpointToPermission(method, fullPath string) (string, string) {
	for _, r := range endpointPermissionTable {
		if r.workspaceOnly && !strings.Contains(fullPath, "/workspaces/") {
			continue
		}
		if !strings.Contains(fullPath, r.pathSubstr) {
			continue
		}
		if !methodIn(r.methods, method) {
			continue
		}
		return r.obj, r.act
	}
	return "", ""
}

