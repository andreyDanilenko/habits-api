package middleware

import (
	"strings"

	"backend/internal/model"
	userRepo "backend/internal/repository/user"
	"backend/pkg/auth/token"

	"github.com/gin-gonic/gin"
)

// OptionalGinAuthMiddleware — как GinAuthMiddleware, но не прерывает запрос при отсутствии токена.
// Если токен валиден — устанавливает user_id, role и user (полная модель) в контекст.
// Используется для публичных эндпоинтов (например, /public/invitations/:token), где текущий пользователь опционален.
func OptionalGinAuthMiddleware(tokenGen *token.Generator, userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		if cookie, err := c.Cookie("access_token"); err == nil {
			tokenString = cookie
		} else if authHeader := c.GetHeader("Authorization"); authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := tokenGen.Validate(tokenString)
		if err != nil {
			c.Next()
			return
		}

		c.Set(GinUserIDKey, claims.UserID)
		c.Set(GinRoleKey, model.UserRole(strings.ToUpper(claims.Role)))

		user, err := userRepo.FindByID(c.Request.Context(), claims.UserID)
		if err == nil && user != nil {
			user.Password = ""
			c.Set("user", user)
		}

		c.Next()
	}
}
