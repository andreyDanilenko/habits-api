package middleware

import (
	"backend/pkg/response"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler(responder *response.Responder) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				responder.InternalServerError(c, "internal server error")
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			lastErr := c.Errors.Last()
			if lastErr != nil {
				if c.Writer.Written() {
					return
				}

				switch lastErr.Type {
				case gin.ErrorTypeBind:
					responder.BadRequest(c, lastErr.Error())
				case gin.ErrorTypePublic:
					responder.WriteError(c, http.StatusBadRequest, lastErr.Error())
				default:
					responder.InternalServerError(c, "internal server error")
				}
				c.Abort()
			}
		}
	}
}
