package response

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse представляет структуру ответа с ошибкой
type ErrorResponse struct {
	Status    string      `json:"status"`
	Error     string      `json:"error"`
	ErrorCode string      `json:"error_code,omitempty"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
}

// WriteError отправляет ответ с ошибкой
func WriteError(c *gin.Context, statusCode int, message string) {
	WriteErrorWithCode(c, statusCode, http.StatusText(statusCode), message, nil)
}

// WriteErrorWithCode отправляет ответ с ошибкой, включая код ошибки и детали
func WriteErrorWithCode(c *gin.Context, statusCode int, errorCode, message string, details interface{}) {
	// Логируем внутренние ошибки (5xx)
	if statusCode >= 500 {
		log.Printf("internal error [%d]: %s - %s", statusCode, errorCode, message)
	}

	response := ErrorResponse{
		Status:    "error",
		Error:     http.StatusText(statusCode),
		ErrorCode: errorCode,
		Message:   message,
		Details:   details,
	}
	c.JSON(statusCode, response)
}

// BadRequest отправляет ответ 400 Bad Request
func BadRequest(c *gin.Context, message string) {
	WriteError(c, http.StatusBadRequest, message)
}

// Unauthorized отправляет ответ 401 Unauthorized
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	WriteError(c, http.StatusUnauthorized, message)
}

// Forbidden отправляет ответ 403 Forbidden
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	WriteError(c, http.StatusForbidden, message)
}

// NotFound отправляет ответ 404 Not Found
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	WriteError(c, http.StatusNotFound, message)
}

// InternalServerError отправляет ответ 500 Internal Server Error
func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	WriteError(c, http.StatusInternalServerError, message)
}

// InternalServerErrorWithDetails отправляет ответ 500 с деталями ошибки
func InternalServerErrorWithDetails(c *gin.Context, message string, err error) {
	details := map[string]string{}
	if err != nil {
		details["error"] = err.Error()
	}
	WriteErrorWithCode(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, details)
}
