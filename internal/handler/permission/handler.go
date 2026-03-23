package permission

import (
	"errors"
	"net/http"

	"backend/internal/middleware"
	"backend/internal/model"
	permRepo "backend/internal/repository/permission"
	permService "backend/internal/service/permission"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   *permService.Service
	responder *response.Responder
	validate  *validator.Validate
}

func NewHandler(service *permService.Service, responder *response.Responder, validate *validator.Validate) *Handler {
	return &Handler{service: service, responder: responder, validate: validate}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/permissions/catalog", h.GetCatalog)
	r.GET("/permissions/system-roles", h.GetSystemRolePermissions)
	r.GET("/roles", h.ListRoles)
	r.POST("/roles", h.CreateRole)
	r.GET("/roles/:roleId", h.GetRole)
	r.PUT("/roles/:roleId", h.UpdateRole)
	r.DELETE("/roles/:roleId", h.DeleteRole)
	r.GET("/roles/:roleId/permissions", h.GetRolePermissions)
	r.GET("/roles/:roleId/object-scopes", h.GetRoleObjectScopes)
	r.PUT("/roles/:roleId/object-scopes", h.PutRoleObjectScopes)
	r.POST("/roles/:roleId/inherit/:parentRoleId", h.AddRoleInheritance)
	r.DELETE("/roles/:roleId/inherit/:parentRoleId", h.RemoveRoleInheritance)
	r.GET("/users/:userId/roles", h.GetUserRoles)
	r.POST("/users/:userId/roles/:roleId", h.AssignRole)
	r.DELETE("/users/:userId/roles/:roleId", h.RemoveRole)
	r.GET("/users/:userId/permissions", h.GetUserPermissions)
	r.POST("/users/:userId/permissions", h.GrantPermission)
	r.DELETE("/users/:userId/permissions/:permissionId", h.RevokePermission)
}

// GetCatalog GET /workspaces/:workspaceId/permissions/catalog
func (h *Handler) GetCatalog(c *gin.Context) {
	catalog, err := h.service.GetCatalog(c.Request.Context())
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get permission catalog")
		return
	}
	// Группируем по модулям для UI (как в спеке)
	modules := make(map[string]map[string][]model.PermissionCatalogItem)
	for _, p := range catalog {
		if modules[p.ModuleCode] == nil {
			modules[p.ModuleCode] = make(map[string][]model.PermissionCatalogItem)
		}
		modules[p.ModuleCode][p.EntityType] = append(modules[p.ModuleCode][p.EntityType], p)
	}
	h.responder.SuccessWithData(c, gin.H{"modules": modules, "catalog": catalog})
}

// GetSystemRolePermissions GET /workspaces/:workspaceId/permissions/system-roles
func (h *Handler) GetSystemRolePermissions(c *gin.Context) {
	perms := h.service.GetSystemRolePermissions(c.Request.Context())
	h.responder.SuccessWithData(c, gin.H{"systemRoles": perms})
}

// ListRoles GET /workspaces/:workspaceId/roles
func (h *Handler) ListRoles(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	if workspaceID == "" {
		h.responder.BadRequest(c, "workspaceId required")
		return
	}
	roles, err := h.service.ListRoles(c.Request.Context(), workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list roles")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"roles": roles})
}

// CreateRole POST /workspaces/:workspaceId/roles
func (h *Handler) CreateRole(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID, _ := middleware.GetUserIDFromGin(c)
	if userID == "" {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	role, err := h.service.CreateRole(c.Request.Context(), workspaceID, req.Name, req.Description, req.Permissions, userID)
	if err != nil {
		if err == permService.ErrRoleSystem {
			h.responder.Forbidden(c, err.Error())
			return
		}
		h.responder.BadRequest(c, err.Error())
		return
	}
	h.responder.Created(c, "Role created", role)
}

// GetRole GET /workspaces/:workspaceId/roles/:roleId
func (h *Handler) GetRole(c *gin.Context) {
	roleID := c.Param("roleId")
	role, err := h.service.GetRole(c.Request.Context(), roleID)
	if err != nil || role == nil {
		h.responder.NotFound(c, "Role not found")
		return
	}
	h.responder.SuccessWithData(c, role)
}

// UpdateRole PUT /workspaces/:workspaceId/roles/:roleId
func (h *Handler) UpdateRole(c *gin.Context) {
	roleID := c.Param("roleId")
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	err := h.service.UpdateRole(c.Request.Context(), roleID, req.Name, req.Description, req.Permissions)
	if err != nil {
		if err == permService.ErrRoleSystem {
			h.responder.Forbidden(c, err.Error())
			return
		}
		h.responder.InternalServerError(c, "Failed to update role")
		return
	}
	role, _ := h.service.GetRole(c.Request.Context(), roleID)
	h.responder.SuccessWithData(c, role)
}

// DeleteRole DELETE /workspaces/:workspaceId/roles/:roleId
func (h *Handler) DeleteRole(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	roleID := c.Param("roleId")
	userID, _ := middleware.GetUserIDFromGin(c)
	assignedBy := ""
	if userID != "" {
		assignedBy = userID
	}
	err := h.service.DeleteRole(c.Request.Context(), workspaceID, roleID, assignedBy)
	if err != nil {
		if err == permService.ErrRoleSystem {
			h.responder.Forbidden(c, err.Error())
			return
		}
		h.responder.NotFound(c, "Role not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// GetRolePermissions GET /workspaces/:workspaceId/roles/:roleId/permissions
func (h *Handler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("roleId")
	perms, err := h.service.GetRolePermissions(c.Request.Context(), roleID)
	if err != nil {
		h.responder.NotFound(c, "Role not found")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"permissions": perms})
}

// GetRoleObjectScopes GET /workspaces/:workspaceId/roles/:roleId/object-scopes
func (h *Handler) GetRoleObjectScopes(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	roleID := c.Param("roleId")
	list, err := h.service.ListRoleObjectScopes(c.Request.Context(), workspaceID, roleID)
	if err != nil {
		if errors.Is(err, permRepo.ErrRoleNotFound) {
			h.responder.NotFound(c, "Role not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to list object scopes")
		return
	}
	m := make(map[string]string, len(list))
	for _, row := range list {
		m[row.ObjectKey] = row.DataScope
	}
	h.responder.SuccessWithData(c, gin.H{"objectScopes": m})
}

// PutRoleObjectScopes PUT /workspaces/:workspaceId/roles/:roleId/object-scopes
// Body: { "objectScopes": { "crm:deal": "owner", "crm:contact": "all" } }
func (h *Handler) PutRoleObjectScopes(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	roleID := c.Param("roleId")
	var req struct {
		ObjectScopes map[string]string `json:"objectScopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if req.ObjectScopes == nil {
		req.ObjectScopes = map[string]string{}
	}
	err := h.service.SetRoleObjectScopes(c.Request.Context(), workspaceID, roleID, req.ObjectScopes)
	if err != nil {
		if errors.Is(err, permRepo.ErrRoleNotFound) {
			h.responder.NotFound(c, "Role not found")
			return
		}
		if errors.Is(err, permService.ErrProtectedRoleObjectScopes) {
			h.responder.Forbidden(c, err.Error())
			return
		}
		h.responder.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// GetUserRoles GET /workspaces/:workspaceId/users/:userId/roles
func (h *Handler) GetUserRoles(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID := c.Param("userId")
	roles, err := h.service.GetUserRolesFull(c.Request.Context(), userID, workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get user roles")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"roles": roles})
}

// AssignRole POST /workspaces/:workspaceId/users/:userId/roles/:roleId
func (h *Handler) AssignRole(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID := c.Param("userId")
	roleID := c.Param("roleId")
	assignedBy, _ := middleware.GetUserIDFromGin(c)
	err := h.service.AssignRole(c.Request.Context(), userID, roleID, workspaceID, assignedBy)
	if err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// RemoveRole DELETE /workspaces/:workspaceId/users/:userId/roles/:roleId
func (h *Handler) RemoveRole(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID := c.Param("userId")
	roleID := c.Param("roleId")
	err := h.service.RemoveRole(c.Request.Context(), userID, roleID, workspaceID)
	if err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// GetUserPermissions GET /workspaces/:workspaceId/users/:userId/permissions
func (h *Handler) GetUserPermissions(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID := c.Param("userId")
	list, err := h.service.GetUserPermissions(c.Request.Context(), userID, workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get user permissions")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"permissions": list})
}

// GrantPermission POST /workspaces/:workspaceId/users/:userId/permissions
func (h *Handler) GrantPermission(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID := c.Param("userId")
	grantedBy, _ := middleware.GetUserIDFromGin(c)
	var req struct {
		PermissionID string  `json:"permissionId" binding:"required"`
		ExpiresAt    *string `json:"expiresAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	err := h.service.GrantPermission(c.Request.Context(), userID, workspaceID, req.PermissionID, grantedBy, req.ExpiresAt)
	if err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// RevokePermission DELETE /workspaces/:workspaceId/users/:userId/permissions/:permissionId
func (h *Handler) RevokePermission(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID := c.Param("userId")
	permissionID := c.Param("permissionId")
	err := h.service.RevokePermission(c.Request.Context(), userID, workspaceID, permissionID)
	if err != nil {
		h.responder.NotFound(c, "Permission not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// AddRoleInheritance POST /workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId
func (h *Handler) AddRoleInheritance(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	childRoleID := c.Param("roleId")
	parentRoleID := c.Param("parentRoleId")
	if workspaceID == "" || childRoleID == "" || parentRoleID == "" {
		h.responder.BadRequest(c, "workspaceId, roleId and parentRoleId are required")
		return
	}
	if err := h.service.AddRoleInheritance(c.Request.Context(), workspaceID, childRoleID, parentRoleID); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// RemoveRoleInheritance DELETE /workspaces/:workspaceId/roles/:roleId/inherit/:parentRoleId
func (h *Handler) RemoveRoleInheritance(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	childRoleID := c.Param("roleId")
	parentRoleID := c.Param("parentRoleId")
	if workspaceID == "" || childRoleID == "" || parentRoleID == "" {
		h.responder.BadRequest(c, "workspaceId, roleId and parentRoleId are required")
		return
	}
	if err := h.service.RemoveRoleInheritance(c.Request.Context(), workspaceID, childRoleID, parentRoleID); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// GetMyPermissions GET /me/permissions?workspaceId=... — права текущего пользователя в workspace (для фронта).
func (h *Handler) GetMyPermissions(c *gin.Context) {
	userID, userRole, ok := middleware.GetAuthFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	workspaceID := c.Query("workspaceId")
	if workspaceID == "" {
		h.responder.BadRequest(c, "workspaceId query parameter required")
		return
	}

	ctx := c.Request.Context()

	var permissions []string
	var roles []string
	// Глобальный ADMIN в PermissionMiddleware имеет полный доступ в любом workspace.
	// Для согласованности фронта возвращаем для него весь каталог прав как эффективные.
	if userRole == model.UserRoleAdmin {
		catalog, err := h.service.GetCatalog(ctx)
		if err != nil {
			h.responder.InternalServerError(c, "Failed to get permission catalog")
			return
		}
		seen := make(map[string]struct{}, len(catalog))
		for _, item := range catalog {
			perm := item.ModuleCode + ":" + item.EntityType + ":" + item.Action
			seen[perm] = struct{}{}
		}
		permissions = make([]string, 0, len(seen))
		for p := range seen {
			permissions = append(permissions, p)
		}
		roles = []string{"ADMIN"}
	} else {
		var err error
		permissions, err = h.service.GetEffectivePermissions(ctx, userID, workspaceID)
		if err != nil {
			h.responder.InternalServerError(c, "Failed to get permissions")
			return
		}
		roles, _ = h.service.GetUserRoles(ctx, userID, workspaceID)
	}
	systemRole := pickSystemRole(roles)

	dataScopes, deptID, hasDept, err := h.service.MeDataScopeContext(ctx, userID, workspaceID, userRole == model.UserRoleAdmin)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get data scopes")
		return
	}
	out := gin.H{
		"permissions": permissions,
		"roles":       roles,
		"systemRole":  systemRole,
		"dataScopes":  dataScopes,
	}
	if hasDept {
		out["departmentId"] = deptID
	}

	h.responder.SuccessWithData(c, out)
}

// pickSystemRole возвращает приоритетную роль workspace (OWNER > ADMIN > MEMBER > GUEST).
// Если только кастомные роли — возвращает первую кастомную роль по имени.
func pickSystemRole(roles []string) string {
	for _, r := range []string{"OWNER", "ADMIN", "MEMBER", "GUEST"} {
		for _, role := range roles {
			if role == r {
				return r
			}
		}
	}
	if len(roles) > 0 {
		return roles[0] // кастомная роль
	}
	return ""
}
