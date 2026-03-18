package admin

import (
	"database/sql"

	"backend/internal/middleware"
	"backend/internal/model"
	userRepo "backend/internal/repository/user"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

const RouteWorkspaces = "/workspaces"
const RouteUsers = "/users"
const RouteUserByID = "/users/:id"
const RouteUserBan = "/users/:id/ban"
const RouteUserUnban = "/users/:id/unban"
const RouteUserLicenses    = "/users/:id/licenses"
const RouteUserLicenseByID = "/users/:id/licenses/:licenseId"

type Handler struct {
	workspaceService *workspaceService.Service
	userRepo         *userRepo.PostgresUserRepository
	responder        *response.Responder
}

func NewHandler(
	workspaceService *workspaceService.Service,
	userRepo *userRepo.PostgresUserRepository,
	responder *response.Responder,
) *Handler {
	return &Handler{
		workspaceService: workspaceService,
		userRepo:         userRepo,
		responder:        responder,
	}
}

const RouteAdminModules       = "/modules"
const RouteWorkspaceByID      = "/workspaces/:workspaceId"
const RouteWorkspaceModules   = "/workspaces/:workspaceId/modules"
const RouteWorkspaceModuleOne = "/workspaces/:workspaceId/modules/:moduleCode"

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET(RouteWorkspaces, h.ListWorkspaces)
	r.GET(RouteWorkspaceByID, h.GetWorkspace)
	r.GET(RouteWorkspaceModules, h.GetWorkspaceModules)
	r.PATCH(RouteWorkspaceModuleOne, h.PatchWorkspaceModule)
	r.GET(RouteUsers, h.ListUsers)
	r.GET(RouteAdminModules, h.ListModules)
	r.DELETE(RouteUserByID, h.DeleteUser)
	r.POST(RouteUserBan, h.BanUser)
	r.POST(RouteUserUnban, h.UnbanUser)
	r.GET(RouteUserLicenses, h.GetUserLicenses)
	r.POST(RouteUserLicenses, h.GrantLicense)
	r.DELETE(RouteUserLicenseByID, h.RevokeLicense)
}

// ListWorkspaces возвращает все workspaces. Вызывать только после RequireAdmin middleware.
func (h *Handler) ListWorkspaces(c *gin.Context) {
	list, err := h.workspaceService.ListAllForAdmin(c.Request.Context())
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list workspaces")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"workspaces": list})
}

// GetWorkspace возвращает workspace по ID (админ).
func (h *Handler) GetWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return
	}
	ws, err := h.workspaceService.GetForAdmin(c.Request.Context(), workspaceID)
	if err != nil {
		if err == workspaceService.ErrWorkspaceNotFound {
			h.responder.NotFound(c, "Workspace not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to get workspace")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"workspace": ws})
}

// GetWorkspaceModules возвращает модули workspace (админ — trial/full).
func (h *Handler) GetWorkspaceModules(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return
	}
	list, err := h.workspaceService.GetWorkspaceModulesForAdmin(c.Request.Context(), workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get workspace modules")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"modules": list})
}

type patchWorkspaceModuleRequest struct {
	Action    string `json:"action" binding:"required"` // extend_trial | add_trial | set_full | set_disabled
	TrialDays *int   `json:"trialDays,omitempty"`
}

// PatchWorkspaceModule — продлить триал, добавить триал или перевести в full (админ).
func (h *Handler) PatchWorkspaceModule(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	moduleCode := c.Param("moduleCode")
	if workspaceID == "" || moduleCode == "" {
		h.responder.BadRequest(c, "Workspace ID and module code required")
		return
	}
	var req patchWorkspaceModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "action required")
		return
	}
	days := 30
	if req.TrialDays != nil && *req.TrialDays > 0 {
		days = *req.TrialDays
	}
	var err error
	switch req.Action {
	case "extend_trial":
		err = h.workspaceService.AdminExtendWorkspaceModuleTrial(c.Request.Context(), workspaceID, moduleCode, days)
	case "add_trial":
		err = h.workspaceService.AdminSetWorkspaceModuleTrial(c.Request.Context(), workspaceID, moduleCode, days)
	case "set_full":
		err = h.workspaceService.AdminSetWorkspaceModuleFull(c.Request.Context(), workspaceID, moduleCode)
	case "set_disabled":
		err = h.workspaceService.AdminSetWorkspaceModuleDisabled(c.Request.Context(), workspaceID, moduleCode)
	default:
		h.responder.BadRequest(c, "action must be extend_trial, add_trial, set_full or set_disabled")
		return
	}
	if err != nil {
		if err == workspaceService.ErrModuleNotFound {
			h.responder.BadRequest(c, "Module not found")
			return
		}
		if err == sql.ErrNoRows {
			h.responder.NotFound(c, "Workspace module not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update workspace module")
		return
	}
	h.responder.SuccessWithMessage(c, "Workspace module updated")
}

// ListUsers возвращает всех пользователей (в т.ч. DELETED) с их workspaces. Только для ADMIN.
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.ListAllIncludingDeleted(c.Request.Context())
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list users")
		return
	}

	type userWithWorkspaces struct {
		ID         string        `json:"id"`
		Email      string        `json:"email"`
		Name       *string       `json:"name,omitempty"`
		Role       string        `json:"role"`
		Status     *string       `json:"status,omitempty"`
		CreatedAt  string        `json:"createdAt"`
		UpdatedAt  string        `json:"updatedAt"`
		Workspaces []interface{} `json:"workspaces"`
	}

	result := make([]userWithWorkspaces, 0, len(users))
	for _, u := range users {
		// Список воркспейсов, в которых пользователь состоит (user_workspaces), без учёта глобального ADMIN
		workspaces, _ := h.workspaceService.List(c.Request.Context(), u.ID, model.UserRoleUser)
		wsList := make([]interface{}, 0, len(workspaces))
		for _, w := range workspaces {
			wsList = append(wsList, map[string]interface{}{
				"id": w.ID, "name": w.Name, "description": w.Description,
				"color": w.Color, "ownerId": w.OwnerID,
				"createdAt": w.CreatedAt, "updatedAt": w.UpdatedAt,
			})
		}
		name := u.Name
		var statusStr *string
		if u.Status != nil {
			s := string(*u.Status)
			statusStr = &s
		}
		result = append(result, userWithWorkspaces{
			ID:         u.ID,
			Email:      u.Email,
			Name:       name,
			Role:       string(u.Role),
			Status:     statusStr,
			CreatedAt:  u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:  u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Workspaces: wsList,
		})
	}
	h.responder.SuccessWithData(c, gin.H{"users": result})
}

// ListModules возвращает все модули из справочника (для админки — выдача лицензий).
func (h *Handler) ListModules(c *gin.Context) {
	list, err := h.workspaceService.ListAllModules(c.Request.Context())
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list modules")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"modules": list})
}

// GetUserLicenses возвращает лицензии пользователя (для админки — просмотр и выдача).
func (h *Handler) GetUserLicenses(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		h.responder.BadRequest(c, "User ID required")
		return
	}
	list, err := h.workspaceService.ListMyLicenses(c.Request.Context(), userID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get user licenses")
		return
	}
	if list == nil {
		list = []model.UserModuleLicense{}
	}
	h.responder.SuccessWithData(c, gin.H{"licenses": list})
}

// RevokeLicense отзывает лицензию у пользователя (только админ).
func (h *Handler) RevokeLicense(c *gin.Context) {
	userID := c.Param("id")
	licenseID := c.Param("licenseId")
	if userID == "" || licenseID == "" {
		h.responder.BadRequest(c, "User ID and License ID required")
		return
	}
	err := h.workspaceService.RevokeLicense(c.Request.Context(), licenseID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.responder.NotFound(c, "License not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to revoke license")
		return
	}
	h.responder.SuccessWithMessage(c, "License revoked successfully")
}

// DeleteUser удаляет пользователя. Только для ADMIN.
// - DELETE /admin/users/:id — мягкое удаление (status = DELETED)
// - DELETE /admin/users/:id?permanent=true — жёсткое удаление (удаление из БД)
// Нельзя удалить себя.
func (h *Handler) DeleteUser(c *gin.Context) {
	currentUserID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	userID := c.Param("id")
	if userID == "" {
		h.responder.BadRequest(c, "User ID required")
		return
	}
	if userID == currentUserID {
		h.responder.BadRequest(c, "Cannot delete your own account")
		return
	}

	permanent := c.Query("permanent") == "true"

	var err error
	if permanent {
		err = h.userRepo.HardDelete(c.Request.Context(), userID)
	} else {
		err = h.userRepo.Delete(c.Request.Context(), userID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			h.responder.NotFound(c, "User not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete user")
		return
	}
	if permanent {
		h.responder.SuccessWithMessage(c, "User permanently deleted")
	} else {
		h.responder.SuccessWithMessage(c, "User deleted successfully")
	}
}

// BanUser банит пользователя (status = BANNED). Нельзя забанить себя.
func (h *Handler) BanUser(c *gin.Context) {
	currentUserID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	userID := c.Param("id")
	if userID == "" {
		h.responder.BadRequest(c, "User ID required")
		return
	}
	if userID == currentUserID {
		h.responder.BadRequest(c, "Cannot ban your own account")
		return
	}
	err := h.userRepo.Ban(c.Request.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.responder.NotFound(c, "User not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to ban user")
		return
	}
	h.responder.SuccessWithMessage(c, "User banned successfully")
}

// UnbanUser снимает бан (status = ACTIVE).
func (h *Handler) UnbanUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		h.responder.BadRequest(c, "User ID required")
		return
	}
	err := h.userRepo.Unban(c.Request.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.responder.NotFound(c, "User not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to unban user")
		return
	}
	h.responder.SuccessWithMessage(c, "User unbanned successfully")
}

type grantLicenseRequest struct {
	ModuleCode   string  `json:"moduleCode" binding:"required"`
	Scope        string  `json:"scope" binding:"required"` // all_workspaces | single_workspace
	WorkspaceID  *string `json:"workspaceId,omitempty"`
}

// GrantLicense выдаёт лицензию пользователю на модуль (только для ADMIN). До момента оплаты — способ дать доступ.
func (h *Handler) GrantLicense(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		h.responder.BadRequest(c, "User ID required")
		return
	}
	var req grantLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "moduleCode and scope are required")
		return
	}
	lic, err := h.workspaceService.GrantLicense(c.Request.Context(), userID, req.ModuleCode, req.Scope, req.WorkspaceID)
	if err != nil {
		if err == workspaceService.ErrModuleNotFound {
			h.responder.BadRequest(c, "Module not found")
			return
		}
		h.responder.BadRequest(c, err.Error())
		return
	}
	h.responder.SuccessWithData(c, gin.H{"license": lic})
}
