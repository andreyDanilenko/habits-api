package invitation

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"backend/internal/model"
	invService "backend/internal/service/invitation"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	RouteCreate       = ""
	RouteList         = ""
	RouteCancel       = "/:invitationId"
	RouteGetByToken   = "/:token"
	RouteAccept       = "/:token/accept"
)

type Handler struct {
	service  InvitationService
	responder *response.Responder
}

type InvitationService interface {
	Create(ctx context.Context, workspaceID, userID string, req model.CreateInvitationRequest) (*model.InvitationResponse, error)
	List(ctx context.Context, workspaceID, userID string, status *string, limit, offset int) ([]model.InvitationResponse, int, error)
	Cancel(ctx context.Context, workspaceID, invitationID, userID string) error
	GetByToken(ctx context.Context, token string, currentUser *model.User) (*model.PublicInvitationResponse, error)
	Accept(ctx context.Context, token string, currentUser *model.User) (*model.AcceptInvitationResponse, error)
}

func NewHandler(service InvitationService, responder *response.Responder) *Handler {
	return &Handler{
		service:  service,
		responder: responder,
	}
}

func (h *Handler) Create(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if uid == "" {
		h.responder.Unauthorized(c, "user not found")
		return
	}

	var req model.CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}

	resp, err := h.service.Create(c.Request.Context(), workspaceID, uid, req)
	if err != nil {
		if errors.Is(err, invService.ErrUserAlreadyInWorkspace) {
			h.responder.BadRequest(c, "User already in workspace")
			return
		}
		h.responder.InternalServerError(c, err.Error())
		return
	}
	h.responder.Created(c, "", resp)
}

func (h *Handler) List(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if uid == "" {
		h.responder.Unauthorized(c, "user not found")
		return
	}

	status := c.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}
	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	list, total, err := h.service.List(c.Request.Context(), workspaceID, uid, statusPtr, limit, offset)
	if err != nil {
		h.responder.Forbidden(c, err.Error())
		return
	}
	h.responder.SuccessWithData(c, gin.H{
		"invitations": list,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

func (h *Handler) Cancel(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	invitationID := c.Param("invitationId")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if uid == "" {
		h.responder.Unauthorized(c, "user not found")
		return
	}

	err := h.service.Cancel(c.Request.Context(), workspaceID, invitationID, uid)
	if err != nil {
		if errors.Is(err, invService.ErrInvitationNotFound) {
			h.responder.NotFound(c, "Invitation not found")
			return
		}
		h.responder.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetByToken(c *gin.Context) {
	token := c.Param("token")
	var currentUser *model.User
	if u, ok := c.Get("user"); ok && u != nil {
		currentUser = u.(*model.User)
	}
	resp, err := h.service.GetByToken(c.Request.Context(), token, currentUser)
	if err != nil {
		if errors.Is(err, invService.ErrInvitationNotFound) || errors.Is(err, invService.ErrInvitationExpired) {
			h.responder.WriteError(c, http.StatusGone, "Invitation not found or expired")
			return
		}
		h.responder.InternalServerError(c, err.Error())
		return
	}
	h.responder.SuccessWithData(c, resp)
}

func (h *Handler) Accept(c *gin.Context) {
	token := c.Param("token")
	var currentUser *model.User
	if u, ok := c.Get("user"); ok && u != nil {
		currentUser = u.(*model.User)
	}
	resp, err := h.service.Accept(c.Request.Context(), token, currentUser)
	if err != nil {
		h.responder.InternalServerError(c, err.Error())
		return
	}
	h.responder.SuccessWithData(c, resp)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST(RouteCreate, h.Create)
	r.GET(RouteList, h.List)
	r.DELETE(RouteCancel, h.Cancel)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET(RouteGetByToken, h.GetByToken)
	r.POST(RouteAccept, h.Accept)
}
