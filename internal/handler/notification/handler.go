package notification

import (
	"errors"
	"fmt"

	"backend/internal/middleware"
	"backend/internal/model"
	notificationRepo "backend/internal/repository/notification"
	notificationService "backend/internal/service/notification"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service  *notificationService.Service
	responder *response.Responder
	validate  *validator.Validate
}

func NewHandler(service *notificationService.Service, responder *response.Responder, validate *validator.Validate) *Handler {
	return &Handler{service: service, responder: responder, validate: validate}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/notifications", h.List)
	r.POST("/notifications", h.Create)
	r.PATCH("/notifications/:id/read", h.MarkRead)
	r.POST("/notifications/mark-all-read", h.MarkAllRead)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	opts := model.NotificationListOpts{
		Channel:     c.Query("channel"),
		UnreadOnly:  c.Query("unreadOnly") == "true" || c.Query("unreadOnly") == "1",
		Limit:   50,
		Offset:  0,
	}
	if l := c.Query("limit"); l != "" {
		if n, err := parseInt(l); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := parseInt(o); err == nil && n >= 0 {
			opts.Offset = n
		}
	}

	list, total, err := h.service.List(c.Request.Context(), userID, opts)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list notifications")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"notifications": list, "total": total})
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	var dto model.CreateNotificationDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if dto.EventKey == "" || dto.Title == "" {
		h.responder.BadRequest(c, "eventKey and title are required")
		return
	}

	n, err := h.service.Upsert(c.Request.Context(), userID, &dto)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to create notification")
		return
	}
	h.responder.SuccessWithData(c, n)
}

func (h *Handler) MarkRead(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	id := c.Param("id")
	if id == "" {
		h.responder.BadRequest(c, "Notification ID required")
		return
	}

	err := h.service.MarkRead(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, notificationRepo.ErrNotFound) {
			h.responder.NotFound(c, "Notification not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to mark as read")
		return
	}
	h.responder.SuccessWithMessage(c, "ok")
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	channel := c.Query("channel")

	err := h.service.MarkAllRead(c.Request.Context(), userID, channel)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to mark all as read")
		return
	}
	h.responder.SuccessWithMessage(c, "ok")
}

// parseInt — простой парсер для query params.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
