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
const RouteUserLicenses = "/users/:id/licenses"

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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET(RouteWorkspaces, h.ListWorkspaces)
	r.GET(RouteUsers, h.ListUsers)
	r.DELETE(RouteUserByID, h.DeleteUser)
	r.POST(RouteUserLicenses, h.GrantLicense)
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

// ListUsers возвращает всех пользователей с их workspaces. Только для ADMIN.
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.ListAll(c.Request.Context())
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

// DeleteUser удаляет пользователя (soft delete). Нельзя удалить себя. Только для ADMIN.
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
	err := h.userRepo.Delete(c.Request.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.responder.NotFound(c, "User not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete user")
		return
	}
	h.responder.SuccessWithMessage(c, "User deleted successfully")
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
