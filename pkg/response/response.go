package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse представляет структуру успешного ответа
type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// WriteJSON отправляет JSON ответ с указанным статус-кодом
func WriteJSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// Success отправляет успешный ответ с данными
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	response := SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	c.JSON(statusCode, response)
}

// SuccessWithData отправляет успешный ответ только с данными (без сообщения)
func SuccessWithData(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, "", data)
}

// SuccessWithMessage отправляет успешный ответ только с сообщением (без данных)
func SuccessWithMessage(c *gin.Context, message string) {
	Success(c, http.StatusOK, message, nil)
}

// Created отправляет ответ со статусом 201 Created
func Created(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusCreated, message, data)
}
