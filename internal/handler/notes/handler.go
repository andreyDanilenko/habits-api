package notes

import (
	"database/sql"
	"errors"

	"backend/internal/middleware"
	"backend/internal/model"
	notesService "backend/internal/service/notes"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	notesSvc     *notesService.Service
	workspaceSvc *workspaceService.Service
	responder    *response.Responder
	validate     *validator.Validate
}

func NewHandler(
	notesSvc *notesService.Service,
	workspaceSvc *workspaceService.Service,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		notesSvc:     notesSvc,
		workspaceSvc: workspaceSvc,
		responder:    responder,
		validate:     validate,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/notes", h.List)
	r.POST("/notes", h.Create)
	r.GET("/notes/:noteId", h.Get)
	r.PUT("/notes/:noteId", h.Update)
	r.DELETE("/notes/:noteId", h.Delete)
}

func (h *Handler) requireWorkspaceAccess(c *gin.Context) (workspaceID, userID string, ok bool) {
	userID, ok = middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return "", "", false
	}
	workspaceID = c.Param("workspaceId")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return "", "", false
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	role := model.UserRoleUser
	if roleVal != nil {
		role = roleVal.(model.UserRole)
	}
	hasAccess, err := h.workspaceSvc.HasAccess(c.Request.Context(), workspaceID, userID, role)
	if err != nil || !hasAccess {
		h.responder.Forbidden(c, "Access denied to this workspace")
		return "", "", false
	}
	return workspaceID, userID, true
}

func (h *Handler) List(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	list, err := h.notesSvc.List(c.Request.Context(), workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list notes")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"notes": list})
}

func (h *Handler) Get(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	noteID := c.Param("noteId")
	n, err := h.notesSvc.Get(c.Request.Context(), workspaceID, noteID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get note")
		return
	}
	if n == nil {
		h.responder.NotFound(c, "Note not found")
		return
	}
	h.responder.SuccessWithData(c, n)
}

func (h *Handler) Create(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "title is required")
		return
	}
	n := &model.Note{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Title:       req.Title,
		Content:     req.Content,
	}
	if err := h.notesSvc.Create(c.Request.Context(), n); err != nil {
		h.responder.InternalServerError(c, "Failed to create note")
		return
	}
	h.responder.SuccessWithData(c, n)
}

func (h *Handler) Update(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	noteID := c.Param("noteId")
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "title is required")
		return
	}
	n := &model.Note{
		ID:          noteID,
		WorkspaceID: workspaceID,
		Title:       req.Title,
		Content:     req.Content,
	}
	if err := h.notesSvc.Update(c.Request.Context(), n); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Note not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update note")
		return
	}
	h.responder.SuccessWithData(c, n)
}

func (h *Handler) Delete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	noteID := c.Param("noteId")
	if err := h.notesSvc.Delete(c.Request.Context(), workspaceID, noteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Note not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete note")
		return
	}
	h.responder.SuccessWithMessage(c, "Note deleted")
}
