package project

import (
	"errors"

	"backend/internal/middleware"
	"backend/internal/model"
	projectService "backend/internal/service/project"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   *projectService.Service
	responder *response.Responder
	validate  *validator.Validate
}

func NewHandler(service *projectService.Service, responder *response.Responder, validate *validator.Validate) *Handler {
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
	r.GET(RouteEntities, h.ListEntityIDs)
	r.POST(RouteAttachEntity, h.AttachEntity)
	r.DELETE(RouteDetachEntity, h.DetachEntity)
	r.GET(RouteEntityProjects, h.GetProjectIDsForEntity)
}

func (h *Handler) getCtx(c *gin.Context) (workspaceID, userID string, role model.UserRole, ok bool) {
	userID, ok = middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return "", "", model.UserRoleUser, false
	}
	workspaceID = c.Param("workspaceId")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return "", "", model.UserRoleUser, false
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	role = model.UserRoleUser
	if roleVal != nil {
		role = roleVal.(model.UserRole)
	}
	return workspaceID, userID, role, true
}

func (h *Handler) List(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	list, err := h.service.List(c.Request.Context(), workspaceID, userID, role)
	if err != nil {
		if errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.Forbidden(c, "Access denied")
			return
		}
		h.responder.InternalServerError(c, "Failed to list projects")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"projects": list})
}

func (h *Handler) Get(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	projectID := c.Param("projectId")
	p, err := h.service.Get(c.Request.Context(), workspaceID, projectID, userID, role)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) || errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.NotFound(c, "Project not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to get project")
		return
	}
	h.responder.SuccessWithData(c, p)
}

func (h *Handler) Create(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	var dto model.CreateProjectDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.responder.BadRequest(c, "Invalid body")
		return
	}
	if err := h.validate.Struct(dto); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	p, err := h.service.Create(c.Request.Context(), workspaceID, userID, role, dto)
	if err != nil {
		if errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.Forbidden(c, "Access denied")
			return
		}
		h.responder.InternalServerError(c, "Failed to create project")
		return
	}
	h.responder.SuccessWithData(c, p)
}

func (h *Handler) Update(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	projectID := c.Param("projectId")
	var dto model.UpdateProjectDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.responder.BadRequest(c, "Invalid body")
		return
	}
	p, err := h.service.Update(c.Request.Context(), workspaceID, projectID, userID, role, dto)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) || errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.NotFound(c, "Project not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update project")
		return
	}
	h.responder.SuccessWithData(c, p)
}

func (h *Handler) Delete(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	projectID := c.Param("projectId")
	err := h.service.Delete(c.Request.Context(), workspaceID, projectID, userID, role)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) || errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.NotFound(c, "Project not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete project")
		return
	}
	h.responder.SuccessWithMessage(c, "Project deleted")
}

func (h *Handler) ListEntityIDs(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	projectID := c.Param("projectId")
	entityType := c.Query("entity_type") // опционально: только crm_deal и т.д.
	ids, err := h.service.ListEntityIDs(c.Request.Context(), workspaceID, projectID, userID, role, entityType)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) || errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.NotFound(c, "Project not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to list entities")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"entityIds": ids})
}

func (h *Handler) AttachEntity(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	projectID := c.Param("projectId")
	var dto model.AttachEntityToProjectDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.responder.BadRequest(c, "Invalid body")
		return
	}
	if err := h.validate.Struct(dto); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	err := h.service.AttachEntity(c.Request.Context(), workspaceID, projectID, userID, role, dto.EntityType, dto.EntityID)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) || errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.NotFound(c, "Project not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to attach entity")
		return
	}
	h.responder.SuccessWithMessage(c, "Entity attached")
}

func (h *Handler) DetachEntity(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	projectID := c.Param("projectId")
	entityType := c.Param("entityType")
	entityID := c.Param("entityId")
	err := h.service.DetachEntity(c.Request.Context(), workspaceID, projectID, userID, role, entityType, entityID)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) || errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.NotFound(c, "Project not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to detach entity")
		return
	}
	h.responder.SuccessWithMessage(c, "Entity detached")
}

func (h *Handler) GetProjectIDsForEntity(c *gin.Context) {
	workspaceID, userID, role, ok := h.getCtx(c)
	if !ok {
		return
	}
	entityType := c.Param("entityType")
	entityID := c.Param("entityId")
	ids, err := h.service.GetProjectIDsForEntity(c.Request.Context(), workspaceID, userID, role, entityType, entityID)
	if err != nil {
		if errors.Is(err, projectService.ErrAccessDenied) {
			h.responder.Forbidden(c, "Access denied")
			return
		}
		h.responder.InternalServerError(c, "Failed to get projects for entity")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"projectIds": ids})
}
