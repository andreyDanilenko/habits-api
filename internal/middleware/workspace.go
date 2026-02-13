package middleware

import (
	"errors"
	"net/http"

	"backend/internal/model"
	workspaceService "backend/internal/service/workspace"

	"github.com/gin-gonic/gin"
)

// WorkspaceMiddleware подставляет workspace_id в контекст.
// Источники (по приоритету): query ?workspace_id=, заголовок X-Workspace-ID, user_preferences, первый доступный.
// Учитывает глобальную роль: ADMIN имеет доступ к любому воркспейсу.
func WorkspaceMiddleware(svc *workspaceService.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := GetUserIDFromGin(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		roleVal, _ := c.Get(GinRoleKey)
		userRole := model.UserRoleUser
		if roleVal != nil {
			userRole = roleVal.(model.UserRole)
		}

		workspaceID := c.Query("workspace_id")
		if workspaceID == "" {
			workspaceID = c.GetHeader("X-Workspace-ID")
		}
		if workspaceID == "" {
			var err error
			workspaceID, err = svc.GetCurrentWorkspace(c.Request.Context(), userID, userRole)
			if err != nil && !errors.Is(err, workspaceService.ErrNoActiveWorkspace) {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get current workspace"})
				return
			}
		}

		if workspaceID != "" {
			hasAccess, err := svc.HasAccess(c.Request.Context(), workspaceID, userID, userRole)
			if err != nil || !hasAccess {
				workspaceID = ""
			}
		}

		c.Set(GinWorkspaceIDKey, workspaceID)
		c.Next()
	}
}

// GetWorkspaceIDFromGin возвращает workspace_id из контекста gin.
func GetWorkspaceIDFromGin(c *gin.Context) (string, bool) {
	v, ok := c.Get(GinWorkspaceIDKey)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
