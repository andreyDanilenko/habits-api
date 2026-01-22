package logger

import (
	"backend/internal/service/logger"
	"backend/pkg/response"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   *logger.Service
	validate  *validator.Validate
	responder *response.Responder
}

func NewHandler(
	service *logger.Service,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		service:   service,
		responder: responder,
		validate:  validate,
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
			h.responder.BadRequest(c, "invalid date format, use YYYY-MM-DD")
			return
		}
	}

	// Получаем логи из БД
	logs, err := h.service.GetLogsByDate(c.Request.Context(), date)
	if err != nil {
		h.responder.InternalServerErrorWithDetails(c, "failed to get logs", err)
		return
	}

	h.responder.SuccessWithData(c, gin.H{
		"date":  date.Format("2006-01-02"),
		"count": len(logs),
		"logs":  logs,
	})
}

// SyncToDB синхронизирует вчерашние логи в БД вручную
func (h *Handler) SyncToDB(c *gin.Context) {
	err := h.service.SyncToDB()
	if err != nil {
		h.responder.InternalServerErrorWithDetails(c, "failed to sync logs", err)
		return
	}

	h.responder.SuccessWithMessage(c, "logs synchronized successfully")
}
