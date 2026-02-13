package middleware

import (
	"backend/internal/model"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func RequireAdmin(responder *response.Responder) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(GinRoleKey)
		if !exists {
			responder.Forbidden(c, "Access denied")
			c.Abort()
			return
		}
		role, ok := roleVal.(model.UserRole)
		if !ok || role != model.UserRoleAdmin {
			responder.Forbidden(c, "Admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}
