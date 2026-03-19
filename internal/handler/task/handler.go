package task

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend/internal/middleware"
	"backend/internal/model"
	taskService "backend/internal/service/task"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const previewTokenTTL = 5 * time.Minute

type previewTokenEntry struct {
	attachmentID string
	workspaceID  string
	expiresAt    time.Time
}

var (
	previewTokens   = make(map[string]previewTokenEntry)
	previewTokensMu sync.RWMutex
)

type Handler struct {
	taskSvc     *taskService.Service
	workspaceSvc *workspaceService.Service
	responder   *response.Responder
	validate    *validator.Validate
	uploadsDir  string
}

func NewHandler(
	taskSvc *taskService.Service,
	workspaceSvc *workspaceService.Service,
	responder *response.Responder,
	validate *validator.Validate,
	uploadsDir string,
) *Handler {
	return &Handler{
		taskSvc:      taskSvc,
		workspaceSvc: workspaceSvc,
		responder:    responder,
		validate:     validate,
		uploadsDir:   uploadsDir,
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
	r.GET("/tasks/:taskId/links", h.ListTaskLinks)
	r.POST("/tasks/:taskId/links", h.AddTaskLink)
	r.DELETE("/tasks/:taskId/links/:linkId", h.DeleteTaskLink)
	r.GET("/tasks/:taskId/attachments", h.ListAttachments)
	r.POST("/tasks/:taskId/attachments", h.CreateAttachment)
	r.GET("/tasks/:taskId/attachments/:attachmentId/download", h.DownloadAttachment)
	r.GET("/tasks/:taskId/attachments/:attachmentId/view", h.ViewAttachment)
	r.GET("/tasks/:taskId/attachments/:attachmentId/preview-token", h.GetPreviewToken)
	r.DELETE("/tasks/:taskId/attachments/:attachmentId", h.DeleteAttachment)
	r.GET("/tasks/:taskId/activities", h.ListTaskActivities)
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
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	var req model.UpdateTaskDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	t, err := h.taskSvc.Update(c.Request.Context(), workspaceID, taskID, userID, req)
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
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	if err := h.taskSvc.Delete(c.Request.Context(), workspaceID, taskID, userID); err != nil {
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
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	t, err := h.taskSvc.Reopen(c.Request.Context(), workspaceID, taskID, userID)
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

func (h *Handler) ListTaskLinks(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	list, err := h.taskSvc.ListTaskLinks(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list task links")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"links": list})
}

func (h *Handler) AddTaskLink(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	var req model.CreateTaskTaskLinkDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	link, err := h.taskSvc.AddTaskLink(c.Request.Context(), workspaceID, taskID, req)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to add task link")
		return
	}
	h.responder.SuccessWithData(c, link)
}

func (h *Handler) DeleteTaskLink(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	linkID := c.Param("linkId")
	if err := h.taskSvc.DeleteTaskLink(c.Request.Context(), workspaceID, linkID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Link not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete task link")
		return
	}
	h.responder.SuccessWithMessage(c, "Link deleted")
}

func (h *Handler) ListAttachments(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	list, err := h.taskSvc.ListAttachments(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list attachments")
		return
	}
	for i := range list {
		list[i].URL = "/api/v1/workspaces/" + workspaceID + "/tasks/" + taskID + "/attachments/" + list[i].ID + "/download"
	}
	h.responder.SuccessWithData(c, gin.H{"attachments": list})
}

func (h *Handler) CreateAttachment(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	file, err := c.FormFile("file")
	if err != nil {
		h.responder.BadRequest(c, "File required")
		return
	}
	src, err := file.Open()
	if err != nil {
		h.responder.InternalServerError(c, "Failed to read file")
		return
	}
	defer src.Close()

	dir := filepath.Join(h.uploadsDir, "tasks", taskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.responder.InternalServerError(c, "Failed to create upload directory")
		return
	}
	ext := filepath.Ext(file.Filename)
	safeName := uuid.New().String() + ext
	destPath := filepath.Join(dir, safeName)
	dst, err := os.Create(destPath)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to save file")
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(destPath)
		h.responder.InternalServerError(c, "Failed to save file")
		return
	}

	relPath := filepath.Join("tasks", taskID, safeName)
	att := &model.TaskAttachment{
		ID:         uuid.New().String(),
		TaskID:     taskID,
		FileName:   file.Filename,
		FilePath:   relPath,
		FileSize:   ptr(int(file.Size)),
		MimeType:   file.Header.Get("Content-Type"),
		UploadedBy: userID,
	}
	if err := h.taskSvc.CreateAttachment(c.Request.Context(), workspaceID, att); err != nil {
		os.Remove(destPath)
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Task not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to create attachment")
		return
	}
	att.URL = "/api/v1/workspaces/" + workspaceID + "/tasks/" + taskID + "/attachments/" + att.ID + "/download"
	h.responder.SuccessWithData(c, att)
}

func (h *Handler) DownloadAttachment(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	attachmentID := c.Param("attachmentId")
	att, err := h.taskSvc.GetAttachment(c.Request.Context(), workspaceID, attachmentID)
	if err != nil || att == nil {
		h.responder.NotFound(c, "Attachment not found")
		return
	}
	fullPath := filepath.Join(h.uploadsDir, att.FilePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		h.responder.NotFound(c, "File not found")
		return
	}
	if c.Query("preview") == "1" && (strings.HasPrefix(att.MimeType, "image/") || isImageExt(att.FileName)) {
		c.Header("Content-Disposition", "inline")
	} else {
		c.Header("Content-Disposition", "attachment; filename=\""+att.FileName+"\"")
	}
	c.File(fullPath)
}

func (h *Handler) GetPreviewToken(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	attachmentID := c.Param("attachmentId")
	att, err := h.taskSvc.GetAttachment(c.Request.Context(), workspaceID, attachmentID)
	if err != nil || att == nil {
		h.responder.NotFound(c, "Attachment not found")
		return
	}
	if att.TaskID != taskID {
		h.responder.NotFound(c, "Attachment not found")
		return
	}
	if !strings.HasPrefix(att.MimeType, "image/") && !isImageExt(att.FileName) {
		h.responder.BadRequest(c, "Not an image")
		return
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		h.responder.InternalServerError(c, "Failed to generate token")
		return
	}
	token := hex.EncodeToString(b)
	previewTokensMu.Lock()
	previewTokens[token] = previewTokenEntry{
		attachmentID: attachmentID,
		workspaceID:  workspaceID,
		expiresAt:    time.Now().Add(previewTokenTTL),
	}
	previewTokensMu.Unlock()
	viewPath := "/api/v1/workspaces/" + workspaceID + "/tasks/" + taskID + "/attachments/" + attachmentID + "/view"
	url := viewPath + "?token=" + token
	h.responder.SuccessWithData(c, gin.H{
		"token":     token,
		"url":      url,
		"expiresIn": int(previewTokenTTL.Seconds()),
	})
}

func (h *Handler) ViewAttachment(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		h.responder.BadRequest(c, "Token required")
		return
	}
	previewTokensMu.RLock()
	entry, ok := previewTokens[token]
	previewTokensMu.RUnlock()
	if !ok {
		h.responder.Unauthorized(c, "Invalid or expired token")
		return
	}
	if time.Now().After(entry.expiresAt) {
		previewTokensMu.Lock()
		delete(previewTokens, token)
		previewTokensMu.Unlock()
		h.responder.Unauthorized(c, "Token expired")
		return
	}
	attachmentID := c.Param("attachmentId")
	workspaceID := c.Param("workspaceId")
	if entry.attachmentID != attachmentID || entry.workspaceID != workspaceID {
		h.responder.Unauthorized(c, "Invalid token")
		return
	}
	att, err := h.taskSvc.GetAttachment(c.Request.Context(), workspaceID, attachmentID)
	if err != nil || att == nil {
		h.responder.NotFound(c, "Attachment not found")
		return
	}
	if !strings.HasPrefix(att.MimeType, "image/") && !isImageExt(att.FileName) {
		h.responder.NotFound(c, "Not an image")
		return
	}
	fullPath := filepath.Join(h.uploadsDir, att.FilePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		h.responder.NotFound(c, "File not found")
		return
	}
	c.Header("Content-Disposition", "inline")
	c.File(fullPath)
}

func isImageExt(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".svg" || ext == ".bmp"
}

func (h *Handler) ListTaskActivities(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	list, total, err := h.taskSvc.ListTaskActivities(c.Request.Context(), workspaceID, taskID, limit, offset)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list activities")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"activities": list, "total": total})
}

func (h *Handler) DeleteAttachment(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	attachmentID := c.Param("attachmentId")
	att, err := h.taskSvc.GetAttachment(c.Request.Context(), workspaceID, attachmentID)
	if err != nil || att == nil {
		h.responder.NotFound(c, "Attachment not found")
		return
	}
	if err := h.taskSvc.DeleteAttachment(c.Request.Context(), workspaceID, attachmentID); err != nil {
		h.responder.InternalServerError(c, "Failed to delete attachment")
		return
	}
	fullPath := filepath.Join(h.uploadsDir, att.FilePath)
	os.Remove(fullPath)
	h.responder.SuccessWithMessage(c, "Attachment deleted")
}

func ptr(i int) *int {
	return &i
}
