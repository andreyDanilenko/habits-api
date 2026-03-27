package chat

import (
	"errors"

	"backend/internal/middleware"
	chatService "backend/internal/service/chat"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	chatSvc      *chatService.Service
	workspaceSvc *workspaceService.Service
	responder    *response.Responder
	validate     *validator.Validate
}

func NewHandler(
	chatSvc *chatService.Service,
	workspaceSvc *workspaceService.Service,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{chatSvc: chatSvc, workspaceSvc: workspaceSvc, responder: responder, validate: validate}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	grp := r.Group("/chat")
	grp.GET("/threads", h.ListThreads)
	grp.POST("/threads/private", h.GetOrCreatePrivateThread)
	grp.GET("/threads/:threadId/messages", h.ListMessages)
	grp.POST("/threads/:threadId/messages", h.SendMessage)
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
	// WorkspacePathMiddleware already checked access; double-check isn't necessary here.
	return workspaceID, userID, true
}

func (h *Handler) ListThreads(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	list, err := h.chatSvc.ListThreads(c.Request.Context(), workspaceID, userID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list chat threads")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"threads": list})
}

type getOrCreatePrivateThreadDto struct {
	OtherUserID string `json:"otherUserId" validate:"required,uuid4"`
}

func (h *Handler) GetOrCreatePrivateThread(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req getOrCreatePrivateThreadDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	thread, _, err := h.chatSvc.GetOrCreatePrivateThread(c.Request.Context(), workspaceID, userID, req.OtherUserID)
	if err != nil {
		if errors.Is(err, chatService.ErrNotFound) || errors.Is(err, chatService.ErrForbidden) {
			h.responder.NotFound(c, "Thread not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to create chat thread")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"thread": thread})
}

func (h *Handler) ListMessages(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	threadID := c.Param("threadId")
	list, err := h.chatSvc.ListMessages(c.Request.Context(), workspaceID, threadID, userID)
	if err != nil {
		if errors.Is(err, chatService.ErrNotFound) || errors.Is(err, chatService.ErrForbidden) {
			h.responder.NotFound(c, "Thread not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to list messages")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"messages": list})
}

type sendMessageDto struct {
	Body string `json:"body" validate:"required,min=1,max=5000"`
}

func (h *Handler) SendMessage(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	threadID := c.Param("threadId")
	var req sendMessageDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}
	msg, err := h.chatSvc.SendMessage(c.Request.Context(), workspaceID, threadID, userID, req.Body)
	if err != nil {
		if errors.Is(err, chatService.ErrNotFound) || errors.Is(err, chatService.ErrForbidden) {
			h.responder.NotFound(c, "Thread not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to send message")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"message": msg})
}

