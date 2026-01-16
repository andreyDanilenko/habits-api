package middleware

import (
	"fmt"
	"strings"

	"backend/internal/model"
	"backend/pkg/auth/token"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	GinUserIDKey = "user_id"
	GinRoleKey   = "role"
)

func GinAuthMiddleware(tokenGen *token.Generator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var tokenFound bool

		// 1. Пробуем получить токен ИЗ КУКИ
		if cookie, err := c.Cookie("access_token"); err == nil {
			tokenString = cookie
			tokenFound = true
			fmt.Println("GinAuthMiddleware: token from cookie")
		} else {
			// 2. Fallback: из заголовка
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				tokenFound = true
				fmt.Println("GinAuthMiddleware: token from header")
			}
		}

		// 3. Если токен не найден - возвращаем 401
		if !tokenFound {
			fmt.Println("GinAuthMiddleware: no token found")
			response.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}

		fmt.Println("GinAuthMiddleware: token found:", tokenString)

		// Валидируем токен
		claims, err := tokenGen.Validate(tokenString)
		if err != nil {
			fmt.Println("GinAuthMiddleware: invalid token:", err)
			response.Unauthorized(c, "Invalid token")
			c.Abort() // ← Прерываем цепочку
			return
		}

		fmt.Println("GinAuthMiddleware: valid token for user:", claims.UserID)

		// Сохраняем в контекст Gin
		c.Set(GinUserIDKey, claims.UserID)
		c.Set(GinRoleKey, model.UserRole(claims.Role))

		c.Next()
	}
}

// Хелпер для хендлеров
func GetUserIDFromGin(c *gin.Context) (string, bool) {
	userID, exists := c.Get(GinUserIDKey)
	if !exists {
		return "", false
	}
	return userID.(string), true
}
