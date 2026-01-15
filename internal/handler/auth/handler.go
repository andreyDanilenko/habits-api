package auth

import (
	"backend/internal/service/auth"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *auth.Service
}

func NewHandler(service *auth.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/login", h.Login)
	r.POST("/register", h.Register)
	r.POST("/logout", h.Logout)
	r.POST("/refresh", h.Refresh)
	r.GET("/me", h.Me)
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
