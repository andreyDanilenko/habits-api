package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (r *Responder) WriteJSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

func (r *Responder) Success(c *gin.Context, statusCode int, message string, data interface{}) {
	response := SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	c.JSON(statusCode, response)
}

func (r *Responder) SuccessWithData(c *gin.Context, data interface{}) {
	r.Success(c, http.StatusOK, "", data)
}

func (r *Responder) SuccessWithMessage(c *gin.Context, message string) {
	r.Success(c, http.StatusOK, message, nil)
}

func (r *Responder) Created(c *gin.Context, message string, data interface{}) {
	r.Success(c, http.StatusCreated, message, data)
}
