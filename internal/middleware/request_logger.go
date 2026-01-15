package middleware

import (
	"backend/internal/service/logger"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger логирует все запросы в файл
func RequestLogger(logService *logger.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Обрабатываем запрос
		c.Next()

		// Формируем строку лога
		duration := time.Since(start)
		timestamp := time.Now()
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		// Формат: [GIN] 2006/01/02 - 15:04:05 | 200 | 4.83725ms | ::1 | GET      "/health"
		logLine := timestamp.Format("[GIN] 2006/01/02 - 15:04:05") +
			" | " + formatInt(statusCode, 3) +
			" | " + formatDuration(duration) +
			" | " + formatString(clientIP, 15) +
			" | " + method +
			"      " + path

		// Записываем в файл синхронно
		logService.WriteLog(logLine)
	}
}

func formatInt(n int, width int) string {
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if s == "" {
		s = "0"
	}
	for len(s) < width {
		s = " " + s
	}
	return s
}

func formatDuration(d time.Duration) string {
	ms := float64(d.Nanoseconds()) / 1000000.0
	if ms < 1 {
		us := float64(d.Nanoseconds()) / 1000.0
		return formatFloat(us, 2) + "µs"
	}
	return formatFloat(ms, 6) + "ms"
}

func formatFloat(f float64, precision int) string {
	s := ""
	n := int(f)
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if s == "" {
		s = "0"
	}
	if precision > 0 {
		frac := f - float64(int(f))
		s += "."
		for i := 0; i < precision; i++ {
			frac *= 10
			s += string(rune('0' + int(frac)%10))
		}
	}
	return s
}

func formatString(s string, width int) string {
	for len(s) < width {
		s += " "
	}
	if len(s) > width {
		return s[:width]
	}
	return s
}
