package logger

import (
	"backend/internal/service/logger"
	"backend/pkg/response"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *logger.Service
}

func NewHandler(service *logger.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.GetLogs)
	r.POST("/sync", h.SyncToDB)
}

func (h *Handler) GetLogs(c *gin.Context) {
	// Получаем дату из query параметра
	dateStr := c.Query("date")
	fmt.Println("123", dateStr)

	var date time.Time
	var err error

	if dateStr == "" {
		// По умолчанию вчерашний день
		date = time.Now().AddDate(0, 0, -1)
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.BadRequest(c, "invalid date format, use YYYY-MM-DD")
			return
		}
	}

	// Получаем логи из БД
	logs, err := h.service.GetLogsByDate(c.Request.Context(), date)
	if err != nil {
		response.InternalServerErrorWithDetails(c, "failed to get logs", err)
		return
	}

	response.SuccessWithData(c, gin.H{
		"date":  date.Format("2006-01-02"),
		"count": len(logs),
		"logs":  logs,
	})
}

// SyncToDB синхронизирует вчерашние логи в БД вручную
func (h *Handler) SyncToDB(c *gin.Context) {
	err := h.service.SyncToDB()
	if err != nil {
		response.InternalServerErrorWithDetails(c, "failed to sync logs", err)
		return
	}

	response.SuccessWithMessage(c, "logs synchronized successfully")
}
