package middleware

import (
	"context"
	"net/http"
	"strings"

	"backend/internal/model"
	"backend/pkg/auth/token"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
	ClaimsKey contextKey = "claims"
)

func AuthMiddleware(tokenGen *token.Generator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string
			var err error

			// 1. Пробуем получить токен ИЗ КУКИ (HTTP-only)
			if cookie, err := r.Cookie("access_token"); err == nil {
				tokenString = cookie.Value
			} else {
				// 2. Fallback: из заголовка (для API/mobile)
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					next.ServeHTTP(w, r)
					return
				}
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// 3. Валидируем токен
			claims, err := tokenGen.Validate(tokenString)
			if err != nil {
				next.ServeHTTP(w, r) // невалидный токен - пропускаем как неавторизованный
				return
			}

			// 4. Добавляем в контекст с ТВОИМИ типами
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, model.UserRole(claims.Role))
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDKey).(string)
	return id, ok
}

func GetUserRole(ctx context.Context) (model.UserRole, bool) {
	role, ok := ctx.Value(RoleKey).(model.UserRole)
	return role, ok
}
