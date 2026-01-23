package habits

import (
	"backend/internal/middleware"
	"backend/internal/model"
	habitsService "backend/internal/service/habits"
	"backend/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct {
	service   *habitsService.Service
	validate  *validator.Validate
	responder *response.Responder
}

func NewHandler(
	service *habitsService.Service,
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
	r.GET(RouteList, h.List)
	r.POST(RouteCreate, h.Create)
	r.GET(RouteGet, h.Get)
	r.PUT(RouteUpdate, h.Update)
	r.DELETE(RouteDelete, h.Delete)
	r.POST(RouteComplete, h.Complete)
	r.POST(RouteToggle, h.Toggle)
	r.GET(RouteStats, h.GetStats)
	r.GET(RouteCompletions, h.GetCompletions)
	r.GET(RouteCalendar, h.GetCalendar)
}

func (h *Handler) requireWorkspace(c *gin.Context) (userID, workspaceID string, ok bool) {
	userID, ok = middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return "", "", false
	}
	workspaceID, ok = middleware.GetWorkspaceIDFromGin(c)
	if !ok || workspaceID == "" {
		h.responder.BadRequest(c, "No workspace selected")
		return "", "", false
	}
	return userID, workspaceID, true
}

func (h *Handler) List(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	list, err := h.service.List(c.Request.Context(), userID, workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list habits")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"habits": list})
}

func (h *Handler) Create(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	var req model.CreateHabitDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}

	habit, err := h.service.Create(c.Request.Context(), req, userID, workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to create habit: "+err.Error())
		return
	}

	h.responder.Created(c, "Habit created", habit)
}

func (h *Handler) Get(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	habit, err := h.service.Get(c.Request.Context(), id, userID, workspaceID)
	if err != nil {
		if err == habitsService.ErrHabitNotFound {
			h.responder.NotFound(c, "Habit not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to get habit")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"habit": habit})
}

func (h *Handler) Update(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	var req model.UpdateHabitDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}

	habit, err := h.service.Update(c.Request.Context(), id, req, userID, workspaceID)
	if err != nil {
		if err == habitsService.ErrHabitNotFound {
			h.responder.NotFound(c, "Habit not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update habit")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"habit": habit})
}

func (h *Handler) Delete(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	err := h.service.Delete(c.Request.Context(), id, userID, workspaceID)
	if err != nil {
		if err == habitsService.ErrHabitNotFound {
			h.responder.NotFound(c, "Habit not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete habit")
		return
	}

	h.responder.SuccessWithMessage(c, "Habit deleted")
}

func (h *Handler) Complete(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	var req struct {
		Date   string `json:"date"`
		Notes  string `json:"notes"`
		Rating int    `json:"rating"`
		Time   string `json:"time"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}
	date, err := parseDate(req.Date)
	if err != nil {
		h.responder.BadRequest(c, "Invalid date")
		return
	}

	// Валидация rating: должен быть от 1 до 5, или 0 (не указан)
	if req.Rating < 0 || req.Rating > 5 {
		h.responder.BadRequest(c, "Rating must be between 0 and 5")
		return
	}

	var timePtr *string
	if req.Time != "" {
		timePtr = &req.Time
	}

	// Если rating = 0, передаем NULL в базу (не указан)
	var ratingValue interface{}
	if req.Rating == 0 {
		ratingValue = nil
	} else {
		ratingValue = req.Rating
	}

	completion, err := h.service.Complete(c.Request.Context(), id, userID, workspaceID, date, req.Notes, ratingValue, timePtr)
	if err != nil {
		if err == habitsService.ErrHabitNotFound {
			h.responder.NotFound(c, "Habit not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to complete habit")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"completion": completion})
}

func (h *Handler) Toggle(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	var req struct {
		Date string `json:"date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}
	date, err := parseDate(req.Date)
	if err != nil {
		h.responder.BadRequest(c, "Invalid date")
		return
	}

	added, completion, err := h.service.Toggle(c.Request.Context(), id, userID, workspaceID, date)
	if err != nil {
		if err == habitsService.ErrHabitNotFound {
			h.responder.NotFound(c, "Habit not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to toggle habit")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"completed": added, "completion": completion})
}

func (h *Handler) GetStats(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), id, userID, workspaceID)
	if err != nil {
		if err == habitsService.ErrHabitNotFound {
			h.responder.NotFound(c, "Habit not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to get stats")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"stats": stats})
}

func (h *Handler) GetCompletions(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	habitID := c.Query("habit_id")
	var habitUUID *uuid.UUID
	if habitID != "" {
		parsed, err := uuid.Parse(habitID)
		if err != nil {
			h.responder.BadRequest(c, "Invalid habit_id")
			return
		}
		habitUUID = &parsed
	}

	start, end, err := parseDateRange(c.Query("start"), c.Query("end"))
	if err != nil {
		h.responder.BadRequest(c, "Invalid start/end date")
		return
	}

	var list []model.HabitCompletion
	if habitUUID != nil {
		// Запрос для конкретной привычки
		completions, err := h.service.GetCompletions(c.Request.Context(), habitUUID.String(), userID, workspaceID, start, end)
		if err != nil {
			if err == habitsService.ErrHabitNotFound {
				h.responder.NotFound(c, "Habit not found")
				return
			}
			h.responder.InternalServerError(c, "Failed to get completions")
			return
		}
		list = completions
	} else {
		// Запрос для всех привычек пользователя в workspace
		completions, err := h.service.GetAllCompletions(c.Request.Context(), userID, workspaceID, start, end)
		if err != nil {
			h.responder.InternalServerError(c, "Failed to get completions")
			return
		}
		list = completions
	}

	h.responder.SuccessWithData(c, gin.H{"completions": list})
}

func (h *Handler) GetCalendar(c *gin.Context) {
	userID, workspaceID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	start, end, err := parseDateRange(c.Query("start"), c.Query("end"))
	if err != nil {
		h.responder.BadRequest(c, "Invalid start/end date")
		return
	}

	cal, err := h.service.GetCalendar(c.Request.Context(), userID, workspaceID, start, end)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get calendar")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"days": cal.Days})
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		// Нормализуем текущую дату до начала дня
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	// Нормализуем дату до начала дня
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, parsed.Location()), nil
}

func parseDateRange(startS, endS string) (start, end time.Time, err error) {
	if startS == "" || endS == "" {
		now := time.Now()
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		// Нормализуем end до начала дня
		end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, end, nil
	}
	start, err = time.Parse("2006-01-02", startS)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	// Нормализуем start до начала дня
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	
	end, err = time.Parse("2006-01-02", endS)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	// Нормализуем end до начала дня
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	
	if end.Before(start) {
		start, end = end, start
	}
	return start, end, nil
}
