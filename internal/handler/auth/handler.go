package auth

import (
	authService "backend/internal/service/auth"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *authService.AuthService
}

func NewHandler(service *authService.AuthService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST(RouteLogin, h.Login)
	r.POST(RouteRegister, h.Register)
	r.POST(RouteLogout, h.Logout)
	r.POST(RouteRefresh, h.Refresh)
	r.GET(RouteMe, h.Me)
}

func (h *Handler) Login(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "login"})
}

func (h *Handler) Register(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "register"})
}

func (h *Handler) Logout(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "logout"})
}

func (h *Handler) Refresh(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "refresh"})
}

func (h *Handler) Me(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "me"})
}
