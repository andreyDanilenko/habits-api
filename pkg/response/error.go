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

func (r *Responder) WriteError(c *gin.Context, statusCode int, message string) {
	simpleCode := simpleErrorCode(statusCode)
	r.WriteErrorWithCode(c, statusCode, simpleCode, message, nil)
}

func (r *Responder) WriteErrorWithCode(c *gin.Context, statusCode int, errorCode, message string, details interface{}) {
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

func (r *Responder) BadRequest(c *gin.Context, message string) {
	r.WriteError(c, http.StatusBadRequest, message)
}

func (r *Responder) Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	r.WriteError(c, http.StatusUnauthorized, message)
}

func (r *Responder) Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	r.WriteError(c, http.StatusForbidden, message)
}

func (r *Responder) NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "resource not found"
	}
	r.WriteError(c, http.StatusNotFound, message)
}

func (r *Responder) InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "internal server error"
	}
	r.WriteError(c, http.StatusInternalServerError, message)
}

func (r *Responder) InternalServerErrorWithDetails(c *gin.Context, message string, err error) {
	details := map[string]string{}
	if err != nil {
		details["error"] = err.Error()
	}
	r.WriteErrorWithCode(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, details)
}
