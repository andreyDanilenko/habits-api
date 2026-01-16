package auth

import (
	"backend/internal/middleware"
	"backend/internal/model"
	authService "backend/internal/service/auth"
	"backend/pkg/http/cookies"
	"backend/pkg/response"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST(RouteLogin, h.Login)
	r.POST(RouteRegister, h.Register)
	r.POST(RouteLogout, h.Logout)
	r.POST(RouteRefresh, h.Refresh)
	r.GET(RouteMe, h.Me)
}

type Handler struct {
	service       *authService.AuthService
	cookieManager *cookies.Manager
	validate      *validator.Validate
}

func NewHandler(service *authService.AuthService, cookieManager *cookies.Manager) *Handler {
	return &Handler{
		service:       service,
		cookieManager: cookieManager,
		validate:      validator.New(),
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	loginResp, err := h.service.Login(c.Request.Context(), req)

	fmt.Println("123123", loginResp)
	if err != nil {
		switch err {
		case authService.ErrInvalidCredentials:
			response.Unauthorized(c, "Invalid email or password")
		default:
			response.InternalServerError(c, "Internal server error")
		}
		return
	}

	expiresAt := time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second)
	h.cookieManager.SetToken(c.Writer, "access_token", loginResp.AccessToken, expiresAt)

	response.SuccessWithData(c, gin.H{
		"user":       loginResp.User,
		"expires_in": loginResp.ExpiresIn,
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Register(c.Request.Context(), req)

	fmt.Println("12313123", user)
	if err != nil {
		switch err {
		case authService.ErrUserExists:
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"user":    user,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	h.cookieManager.Delete(c.Writer, "access_token")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out",
	})
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.service.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
