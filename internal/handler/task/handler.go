package task

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"backend/internal/middleware"
	"backend/internal/model"
	taskService "backend/internal/service/task"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	taskSvc     *taskService.Service
	workspaceSvc *workspaceService.Service
	responder   *response.Responder
	validate    *validator.Validate
}

func NewHandler(
	taskSvc *taskService.Service,
	workspaceSvc *workspaceService.Service,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		taskSvc:      taskSvc,
		workspaceSvc: workspaceSvc,
		responder:    responder,
		validate:     validate,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/tasks", h.List)
	r.POST("/tasks", h.Create)
	r.GET("/tasks/:taskId", h.Get)
	r.PATCH("/tasks/:taskId", h.Update)
	r.DELETE("/tasks/:taskId", h.Delete)
	r.POST("/tasks/:taskId/complete", h.Complete)
	r.POST("/tasks/:taskId/reopen", h.Reopen)
	r.GET("/tasks/:taskId/comments", h.ListComments)
	r.POST("/tasks/:taskId/comments", h.CreateComment)
	r.PATCH("/tasks/:taskId/comments/:commentId", h.UpdateComment)
	r.DELETE("/tasks/:taskId/comments/:commentId", h.DeleteComment)
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
	filters := &model.TaskListFilters{
		Status:       c.Query("status"),
		Priority:     c.Query("priority"),
		Type:         c.Query("type"),
		AssigneeID:   c.Query("assigneeId"),
		EntityType:   c.Query("entityType"),
		EntityID:     c.Query("entityId"),
		ParentID:     c.Query("parentId"),
		OverdueOnly:  c.Query("overdue") == "true" || c.Query("overdue") == "1",
		Search:       strings.TrimSpace(c.Query("search")),
	}
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filters.Page = p
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filters.Limit = l
		}
	}
	list, err := h.taskSvc.List(c.Request.Context(), workspaceID, filters)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list tasks")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"tasks": list})
}

func (h *Handler) Get(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	t, err := h.taskSvc.Get(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get task")
		return
	}
	if t == nil {
		h.responder.NotFound(c, "Task not found")
		return
	}
	h.responder.SuccessWithData(c, t)
}

func (h *Handler) Create(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req model.CreateTaskDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	t, err := h.taskSvc.Create(c.Request.Context(), workspaceID, userID, req)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to create task")
		return
	}
	h.responder.SuccessWithData(c, t)
}

func (h *Handler) Update(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	var req model.UpdateTaskDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	t, err := h.taskSvc.Update(c.Request.Context(), workspaceID, taskID, req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Task not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update task")
		return
	}
	h.responder.SuccessWithData(c, t)
}

func (h *Handler) Delete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	if err := h.taskSvc.Delete(c.Request.Context(), workspaceID, taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Task not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete task")
		return
	}
	h.responder.SuccessWithMessage(c, "Task deleted")
}

func (h *Handler) Complete(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	var req model.CompleteTaskDto
	_ = c.ShouldBindJSON(&req)
	t, err := h.taskSvc.Complete(c.Request.Context(), workspaceID, taskID, userID, req.Note)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to complete task")
		return
	}
	if t == nil {
		h.responder.NotFound(c, "Task not found")
		return
	}
	h.responder.SuccessWithData(c, t)
}

func (h *Handler) Reopen(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	t, err := h.taskSvc.Reopen(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to reopen task")
		return
	}
	if t == nil {
		h.responder.NotFound(c, "Task not found or not completed")
		return
	}
	h.responder.SuccessWithData(c, t)
}

func (h *Handler) ListComments(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	list, err := h.taskSvc.ListComments(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list comments")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"comments": list})
}

func (h *Handler) CreateComment(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	var req model.CreateTaskCommentDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	comment, err := h.taskSvc.CreateComment(c.Request.Context(), workspaceID, taskID, userID, req.Body, req.ParentID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to create comment")
		return
	}
	if comment == nil {
		h.responder.NotFound(c, "Task not found")
		return
	}
	h.responder.SuccessWithData(c, comment)
}

func (h *Handler) UpdateComment(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	commentID := c.Param("commentId")
	var req model.UpdateTaskCommentDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	comment, err := h.taskSvc.UpdateComment(c.Request.Context(), workspaceID, commentID, userID, req.Body)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Comment not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update comment")
		return
	}
	if comment == nil {
		h.responder.NotFound(c, "Comment not found")
		return
	}
	h.responder.SuccessWithData(c, comment)
}

func (h *Handler) DeleteComment(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	commentID := c.Param("commentId")
	if err := h.taskSvc.DeleteComment(c.Request.Context(), workspaceID, commentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Comment not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete comment")
		return
	}
	h.responder.SuccessWithMessage(c, "Comment deleted")
}
