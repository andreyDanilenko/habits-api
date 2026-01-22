package middleware

import (
	"errors"
	"net/http"

	workspaceService "backend/internal/service/workspace"

	"github.com/gin-gonic/gin"
)

// WorkspaceMiddleware подставляет workspace_id в контекст.
// Источники (по приоритету): query ?workspace_id=, заголовок X-Workspace-ID, user_preferences, первый доступный.
func WorkspaceMiddleware(svc *workspaceService.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := GetUserIDFromGin(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		workspaceID := c.Query("workspace_id")
		if workspaceID == "" {
			workspaceID = c.GetHeader("X-Workspace-ID")
		}
		if workspaceID == "" {
			var err error
			workspaceID, err = svc.GetCurrentWorkspace(c.Request.Context(), userID)
			if err != nil && !errors.Is(err, workspaceService.ErrNoActiveWorkspace) {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get current workspace"})
				return
			}
		}

		if workspaceID != "" {
			hasAccess, err := svc.HasAccess(c.Request.Context(), workspaceID, userID)
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
