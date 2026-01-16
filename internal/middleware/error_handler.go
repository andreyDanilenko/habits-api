package middleware

import (
	"backend/pkg/response"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Восстанавливаемся от паники
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)

				response.InternalServerError(c, "internal server error")
				c.Abort()
			}
		}()

		c.Next()

		// Проверяем, есть ли ошибки в контексте
		if len(c.Errors) > 0 {
			lastErr := c.Errors.Last()
			if lastErr != nil {
				// Если ошибка уже была обработана, не делаем ничего
				if c.Writer.Written() {
					return
				}

				// Обрабатываем ошибку в зависимости от её типа
				switch lastErr.Type {
				case gin.ErrorTypeBind:
					response.BadRequest(c, lastErr.Error())
				case gin.ErrorTypePublic:
					response.WriteError(c, http.StatusBadRequest, lastErr.Error())
				default:
					response.InternalServerError(c, "internal server error")
				}
				c.Abort()
			}
		}
	}
}
