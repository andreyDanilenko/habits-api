package response

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Status    string      `json:"status"`
	Error     string      `json:"error"`
	ErrorCode string      `json:"error_code,omitempty"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
}

func simpleErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest: // 400
		return "BAD_REQUEST"
	case http.StatusUnauthorized: // 401
		return "UNAUTHORIZED"
	case http.StatusForbidden: // 403
		return "FORBIDDEN"
	case http.StatusNotFound: // 404
		return "NOT_FOUND"
	case http.StatusInternalServerError: // 500
		return "INTERNAL_ERROR"
	default:
		return ""
	}
}

func WriteError(c *gin.Context, statusCode int, message string) {
	simpleCode := simpleErrorCode(statusCode)
	WriteErrorWithCode(c, statusCode, simpleCode, message, nil)
}
func WriteErrorWithCode(c *gin.Context, statusCode int, errorCode, message string, details interface{}) {
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

func BadRequest(c *gin.Context, message string) {
	WriteError(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	WriteError(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	WriteError(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	WriteError(c, http.StatusNotFound, message)
}

func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	WriteError(c, http.StatusInternalServerError, message)
}

func InternalServerErrorWithDetails(c *gin.Context, message string, err error) {
	details := map[string]string{}
	if err != nil {
		details["error"] = err.Error()
	}
	WriteErrorWithCode(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, details)
}
