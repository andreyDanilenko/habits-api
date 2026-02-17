package journal

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"backend/internal/middleware"
	"backend/internal/model"
	journalService "backend/internal/service/journal"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service      *journalService.Service
	workspaceSvc *workspaceService.Service
	responder    *response.Responder
	validate     *validator.Validate
}

func NewHandler(
	service *journalService.Service,
	workspaceSvc *workspaceService.Service,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		service:      service,
		workspaceSvc: workspaceSvc,
		responder:    responder,
		validate:     validate,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	journal := r.Group("/journal")
	{
		journal.GET(RouteList, h.List)
		journal.POST(RouteCreate, h.Create)
		journal.GET(RouteGet, h.Get)
		journal.PUT(RouteUpdate, h.Update)
		journal.DELETE(RouteDelete, h.Delete)
	}
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
	var date *time.Time
	if dateStr := c.Query("date"); dateStr != "" {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.responder.BadRequest(c, "Invalid date format. Use YYYY-MM-DD")
			return
		}
		date = &t
	}
	list, err := h.service.List(c.Request.Context(), workspaceID, date)
	if err != nil {
		log.Printf("[journal] List failed: %v", err)
		h.responder.InternalServerError(c, "Failed to list journal entries")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"entries": list})
}

func (h *Handler) Get(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	entryID := c.Param("entryId")
	entry, err := h.service.Get(c.Request.Context(), workspaceID, entryID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get entry")
		return
	}
	if entry == nil {
		h.responder.NotFound(c, "Entry not found")
		return
	}
	h.responder.SuccessWithData(c, entry)
}

func (h *Handler) Create(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req model.CreateJournalEntryDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}
	entry, err := h.service.Create(c.Request.Context(), workspaceID, userID, req)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to create entry")
		return
	}
	h.responder.Created(c, "Entry created", entry)
}

func (h *Handler) Update(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	entryID := c.Param("entryId")
	var req model.UpdateJournalEntryDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}
	entry, err := h.service.Update(c.Request.Context(), workspaceID, entryID, req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Entry not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update entry")
		return
	}
	h.responder.SuccessWithData(c, entry)
}

func (h *Handler) Delete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	entryID := c.Param("entryId")
	if err := h.service.Delete(c.Request.Context(), workspaceID, entryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Entry not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete entry")
		return
	}
	h.responder.SuccessWithMessage(c, "Entry deleted")
}
