package workspace

import (
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
	r.GET(RouteGet, h.Get)
	r.PUT(RouteUpdate, h.Update)
	r.DELETE(RouteDelete, h.Delete)
	r.GET(RouteMembers, h.GetMembers)
	r.POST(RouteSwitch, h.Switch)
	r.GET(RouteModules, h.GetModules)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	workspaces, err := h.service.List(c.Request.Context(), userID)
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

func (h *Handler) Switch(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "switch workspace"})
}

func (h *Handler) GetModules(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get modules"})
}
