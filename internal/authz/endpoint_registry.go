package authz

import (
	"net/http"
	"strings"
)

// EndpointRule задаёт маппинг (path substring, method) → (obj, act) для permission_catalog.
type EndpointRule struct {
	PathSubstr    string
	WorkspaceOnly bool
	Methods       []string
	Object        string
	Action        string
}

var endpointRules []EndpointRule

// RegisterEndpointRules добавляет правила в реестр. Вызывается при инициализации модулей.
func RegisterEndpointRules(rules []EndpointRule) {
	endpointRules = append(endpointRules, rules...)
}

func methodIn(list []string, m string) bool {
	for _, x := range list {
		if x == m {
			return true
		}
	}
	return false
}

// MapEndpointToPermission маппит HTTP-метод и путь на (object, action) из permission_catalog.
func MapEndpointToPermission(method, fullPath string) (string, string) {
	for _, r := range endpointRules {
		if r.WorkspaceOnly && !strings.Contains(fullPath, "/workspaces/") {
			continue
		}
		if !strings.Contains(fullPath, r.PathSubstr) {
			continue
		}
		if !methodIn(r.Methods, method) {
			continue
		}
		return r.Object, r.Action
	}
	return "", ""
}

func init() {
	RegisterEndpointRules(workspaceRules())
	RegisterEndpointRules(crmRules())
	RegisterEndpointRules(habitsRules())
	RegisterEndpointRules(projectsRules())
	RegisterEndpointRules(tasksRules())
}

func workspaceRules() []EndpointRule {
	return []EndpointRule{
		{"/members", true, []string{http.MethodGet, http.MethodPost}, "workspace:member", "invite"},
		{"/members", true, []string{http.MethodDelete}, "workspace:member", "remove"},
		{"/invitations", true, []string{http.MethodGet, http.MethodPost}, "workspace:member", "invite"},
		{"/invitations", true, []string{http.MethodDelete}, "workspace:member", "remove"},
		{"/permissions/system-roles", true, []string{http.MethodGet}, "workspace:module", "read"},
		{"/modules", true, []string{http.MethodGet}, "workspace:module", "read"},
		{"/modules", true, []string{http.MethodPost, http.MethodDelete}, "workspace:module", "manage"},
		{"/roles", true, []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}, "workspace:role", "manage"},
	}
}

func crmRules() []EndpointRule {
	return []EndpointRule{
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
	}
}

func habitsRules() []EndpointRule {
	return []EndpointRule{
		{"/habits/activities", false, []string{http.MethodGet}, "habits:habit", "read"},
		{"/habits/habits", false, []string{http.MethodGet}, "habits:habit", "read"},
		{"/habits/habits", false, []string{http.MethodPost}, "habits:habit", "create"},
		{"/habits/habits", false, []string{http.MethodPut, http.MethodPatch}, "habits:habit", "update"},
		{"/habits/habits", false, []string{http.MethodDelete}, "habits:habit", "delete"},
		{"/habits/journal", false, []string{http.MethodGet}, "habits:journal", "read"},
		{"/habits/journal", false, []string{http.MethodPost}, "habits:journal", "create"},
		{"/habits/journal", false, []string{http.MethodPut, http.MethodPatch}, "habits:journal", "update"},
		{"/habits/journal", false, []string{http.MethodDelete}, "habits:journal", "delete"},
	}
}

func projectsRules() []EndpointRule {
	return []EndpointRule{
		{"/projects", false, []string{http.MethodGet}, "projects:project", "read"},
		{"/projects", false, []string{http.MethodPost}, "projects:project", "create"},
		{"/projects", false, []string{http.MethodPut, http.MethodPatch}, "projects:project", "update"},
		{"/projects", false, []string{http.MethodDelete}, "projects:project", "delete"},
	}
}

func tasksRules() []EndpointRule {
	return []EndpointRule{
		{"/tasks/", false, []string{http.MethodGet}, "tasks:task", "read"},      // GET /tasks/:id/comments — активности доступны всем, кто может смотреть задачу
		{"/tasks/", false, []string{http.MethodPost}, "tasks:task", "update"},  // POST /tasks/:id/complete, reopen, comments
		{"/tasks", false, []string{http.MethodGet}, "tasks:task", "read"},
		{"/tasks", false, []string{http.MethodPost}, "tasks:task", "create"},
		{"/tasks", false, []string{http.MethodPut, http.MethodPatch}, "tasks:task", "update"},
		{"/tasks", false, []string{http.MethodDelete}, "tasks:task", "delete"},
	}
}
