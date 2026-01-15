package handler

import (
	"backend/internal/router"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RegisterRoutes(r *router.Router) {
	// Health check
	r.GET("/health", HealthCheck)

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/ping", Ping)
	}
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"service": "backend",
	})
}

func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

