package workspace

import (
	"database/sql"
	"errors"

	"backend/internal/middleware"
	"backend/internal/model"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   *workspaceService.Service
	validate  *validator.Validate
	responder *response.Responder
}

func NewHandler(
	service *workspaceService.Service,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		service:   service,
		responder: responder,
		validate:  validate,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET(RouteList, h.List)
	r.POST(RouteCreate, h.Create)
	r.GET(RouteCurrent, h.GetCurrent)
	r.GET(RouteMyLicenses, h.GetMyLicenses)
	r.GET(RouteGet, h.Get)
	r.PUT(RouteUpdate, h.Update)
	r.DELETE(RouteDelete, h.Delete)
	r.GET(RouteMembers, h.GetMembers)
	r.POST(RouteSwitch, h.Switch)
	r.GET(RouteModules, h.GetModules)
	r.POST(RouteModules, h.EnableModule)
	r.DELETE(RouteModuleOne, h.DisableModule)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	userRole, _ := c.Get(middleware.GinRoleKey)
	role := model.UserRoleUser
	if userRole != nil {
		role = userRole.(model.UserRole)
	}

	workspaces, err := h.service.List(c.Request.Context(), userID, role)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list workspaces")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"workspaces": workspaces})
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	var req model.CreateWorkspaceDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}

	workspace, err := h.service.Create(c.Request.Context(), req, userID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to create workspace")
		return
	}

	h.responder.Created(c, "Workspace created successfully", workspace)
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	role, _ := c.Get(middleware.GinRoleKey)
	userRole := role.(model.UserRole)

	workspaceID := c.Param("id")
	workspace, err := h.service.Get(c.Request.Context(), workspaceID, userID, userRole)
	if err != nil {
		switch err {
		case workspaceService.ErrWorkspaceNotFound:
			h.responder.NotFound(c, "Workspace not found")
		case workspaceService.ErrAccessDenied:
			h.responder.Forbidden(c, "Access denied")
		default:
			h.responder.InternalServerError(c, "Failed to get workspace")
		}
		return
	}

	h.responder.SuccessWithData(c, gin.H{"workspace": workspace})
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	role, _ := c.Get(middleware.GinRoleKey)
	userRole := role.(model.UserRole)

	workspaceID := c.Param("id")
	var req model.UpdateWorkspaceDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}

	workspace, err := h.service.Update(c.Request.Context(), workspaceID, req, userID, userRole)
	if err != nil {
		switch err {
		case workspaceService.ErrWorkspaceNotFound:
			h.responder.NotFound(c, "Workspace not found")
		case workspaceService.ErrAccessDenied:
			h.responder.Forbidden(c, "Access denied")
		default:
			h.responder.InternalServerError(c, "Failed to update workspace")
		}
		return
	}

	h.responder.SuccessWithData(c, gin.H{"workspace": workspace})
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	role, _ := c.Get(middleware.GinRoleKey)
	userRole := role.(model.UserRole)

	workspaceID := c.Param("id")
	err := h.service.Delete(c.Request.Context(), workspaceID, userID, userRole)
	if err != nil {
		switch err {
		case workspaceService.ErrWorkspaceNotFound:
			h.responder.NotFound(c, "Workspace not found")
		case workspaceService.ErrAccessDenied:
			h.responder.Forbidden(c, "Access denied")
		default:
			h.responder.InternalServerError(c, "Failed to delete workspace")
		}
		return
	}

	h.responder.SuccessWithMessage(c, "Workspace deleted successfully")
}

func (h *Handler) GetMembers(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get members"})
}

func (h *Handler) GetCurrent(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	userRole := model.UserRoleUser
	if roleVal != nil {
		userRole = roleVal.(model.UserRole)
	}

	workspaceID, err := h.service.GetCurrentWorkspace(c.Request.Context(), userID, userRole)
	if err != nil {
		if err == workspaceService.ErrNoActiveWorkspace {
			h.responder.NotFound(c, "No workspace selected")
			return
		}
		h.responder.InternalServerError(c, "Failed to get current workspace")
		return
	}

	ws, err := h.service.Get(c.Request.Context(), workspaceID, userID, userRole)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get workspace")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"workspace": ws})
}

func (h *Handler) Switch(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	userRole := model.UserRoleUser
	if roleVal != nil {
		userRole = roleVal.(model.UserRole)
	}

	workspaceID := c.Param("id")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return
	}

	hasAccess, err := h.service.HasAccess(c.Request.Context(), workspaceID, userID, userRole)
	if err != nil || !hasAccess {
		h.responder.Forbidden(c, "Access denied to this workspace")
		return
	}

	err = h.service.SetCurrentWorkspace(c.Request.Context(), userID, workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to switch workspace")
		return
	}

	h.responder.SuccessWithMessage(c, "Workspace switched successfully")
}

func (h *Handler) GetMyLicenses(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	list, err := h.service.ListMyLicenses(c.Request.Context(), userID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get licenses")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"licenses": list})
}

func (h *Handler) GetModules(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	userRole := model.UserRoleUser
	if roleVal != nil {
		userRole = roleVal.(model.UserRole)
	}

	workspaceID := c.Param("id")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return
	}

	list, err := h.service.GetWorkspaceModules(c.Request.Context(), workspaceID, userID, userRole)
	if err != nil {
		if err == workspaceService.ErrAccessDenied {
			h.responder.Forbidden(c, "Access denied to this workspace")
			return
		}
		h.responder.InternalServerError(c, "Failed to get workspace modules")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"modules": list})
}

type enableModuleRequest struct {
	ModuleCode string `json:"moduleCode" binding:"required"`
}

func (h *Handler) EnableModule(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	userRole := model.UserRoleUser
	if roleVal != nil {
		userRole = roleVal.(model.UserRole)
	}

	workspaceID := c.Param("id")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return
	}

	var req enableModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "moduleCode is required")
		return
	}

	err := h.service.EnableModule(c.Request.Context(), workspaceID, userID, userRole, req.ModuleCode)
	if err != nil {
		if err == workspaceService.ErrAccessDenied {
			h.responder.Forbidden(c, "Access denied to this workspace")
			return
		}
		if err == workspaceService.ErrModuleNotFound {
			h.responder.BadRequest(c, "Module not found")
			return
		}
		if err == workspaceService.ErrLicenseRequired {
			h.responder.Forbidden(c, "License required: purchase the module or request an admin grant")
			return
		}
		h.responder.InternalServerError(c, "Failed to enable module")
		return
	}

	h.responder.SuccessWithMessage(c, "Module enabled successfully")
}

func (h *Handler) DisableModule(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	userRole := model.UserRoleUser
	if roleVal != nil {
		userRole = roleVal.(model.UserRole)
	}

	workspaceID := c.Param("id")
	moduleCode := c.Param("moduleCode")
	if workspaceID == "" || moduleCode == "" {
		h.responder.BadRequest(c, "Workspace ID and module code required")
		return
	}

	err := h.service.DisableModule(c.Request.Context(), workspaceID, userID, userRole, moduleCode)
	if err != nil {
		if err == workspaceService.ErrAccessDenied {
			h.responder.Forbidden(c, "Access denied to this workspace")
			return
		}
		if err == workspaceService.ErrModuleNotFound {
			h.responder.BadRequest(c, "Module not found")
			return
		}
		// Модуль уже не был включён в этом workspace — считаем отключение успешным (идемпотентность)
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.SuccessWithMessage(c, "Module disabled successfully")
			return
		}
		h.responder.InternalServerError(c, "Failed to disable module")
		return
	}

	h.responder.SuccessWithMessage(c, "Module disabled successfully")
}
