package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

var allowedOrigins = []string{
	"http://localhost:3000",
	"http://192.168.0.5:3000",
	"http://localhost:3000/", // Добавить с слешем для точного совпадения
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		fmt.Println("Request from origin:", origin)
		// Проверяем разрешенные origins
		if origin != "" {
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				// Убираем trailing slash для сравнения
				cleanOrigin := strings.TrimRight(origin, "/")
				cleanAllowed := strings.TrimRight(allowedOrigin, "/")

				if cleanOrigin == cleanAllowed {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
					allowed = true
					break
				}
			}

			// Если origin не разрешен, не устанавливаем CORS заголовки
			if !allowed {
				c.Next()
				return
			}
		}

		// Устанавливаем остальные заголовки
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Workspace-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Credentials")

		// Handle preflight
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
