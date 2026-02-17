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

	habits := r.Group("/habits")
	{
		habits.GET(RouteList, h.List)
		habits.POST(RouteCreate, h.Create)
		habits.GET(RouteGet, h.Get)
		habits.PUT(RouteUpdate, h.Update)
		habits.DELETE(RouteDelete, h.Delete)
		habits.POST(RouteComplete, h.Complete)
		habits.POST(RouteToggle, h.Toggle)
		habits.GET(RouteStats, h.GetStats)
		habits.GET(RouteCompletions, h.GetCompletions)
		habits.GET(RouteCalendar, h.GetCalendar)
	}
}

func (h *Handler) requireWorkspace(c *gin.Context) (userID string, ok bool) {
	userID, ok = middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return "", false
	}

	return userID, true
}

func (h *Handler) List(c *gin.Context) {
	_, ok := h.requireWorkspace(c)
	if !ok {
		return
	}
	workspaceIDParam := c.Param("workspaceId")

	var targetDate *time.Time
	if dateStr := c.Query("date"); dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.responder.BadRequest(c, "Invalid date format. Use YYYY-MM-DD")
			return
		}
		targetDate = &parsedDate
	}

	list, err := h.service.List(c.Request.Context(), workspaceIDParam, targetDate)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list habits")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"habits": list})
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}
	workspaceIDParam := c.Param("workspaceId")

	var req model.CreateHabitDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}

	habit, err := h.service.Create(c.Request.Context(), req, userID, workspaceIDParam)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to create habit: "+err.Error())
		return
	}

	h.responder.Created(c, "Habit created", habit)
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	habitsID := c.Param("habitsId")
	workspaceIDParam := c.Param("workspaceId")

	if _, err := uuid.Parse(habitsID); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	habit, err := h.service.Get(c.Request.Context(), habitsID, userID, workspaceIDParam)
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
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	workspaceIDParam := c.Param("workspaceId")
	habitID := c.Param("habitId")

	if _, err := uuid.Parse(habitID); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	var req model.UpdateHabitDto
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}

	habit, err := h.service.Update(c.Request.Context(), habitID, req, userID, workspaceIDParam)
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
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	workspaceIDParam := c.Param("workspaceId")
	habitID := c.Param("habitId")
	if _, err := uuid.Parse(habitID); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	err := h.service.Delete(c.Request.Context(), habitID, userID, workspaceIDParam)
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
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	workspaceIDParam := c.Param("workspaceId")
	habitID := c.Param("habitId")
	if _, err := uuid.Parse(habitID); err != nil {
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

	if req.Rating < 0 || req.Rating > 5 {
		h.responder.BadRequest(c, "Rating must be between 0 and 5")
		return
	}

	var timePtr *string
	if req.Time != "" {
		timePtr = &req.Time
	}

	var ratingValue interface{}
	if req.Rating == 0 {
		ratingValue = nil
	} else {
		ratingValue = req.Rating
	}

	completion, err := h.service.Complete(c.Request.Context(), habitID, userID, workspaceIDParam, date, req.Notes, ratingValue, timePtr)
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
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	workspaceIDParam := c.Param("workspaceId")
	habitID := c.Param("habitId")
	if _, err := uuid.Parse(habitID); err != nil {
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

	added, completion, err := h.service.Toggle(c.Request.Context(), habitID, userID, workspaceIDParam, date)
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
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	workspaceIDParam := c.Param("workspaceId")
	habitID := c.Param("habitId")
	if _, err := uuid.Parse(habitID); err != nil {
		h.responder.BadRequest(c, "Invalid habit ID")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), habitID, userID, workspaceIDParam)
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
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	workspaceIDParam := c.Param("workspaceId")
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
		completions, err := h.service.GetCompletions(c.Request.Context(), habitUUID.String(), userID, workspaceIDParam, start, end)
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
		completions, err := h.service.GetAllCompletions(c.Request.Context(), userID, workspaceIDParam, start, end)
		if err != nil {
			h.responder.InternalServerError(c, "Failed to get completions")
			return
		}
		list = completions
	}

	h.responder.SuccessWithData(c, gin.H{"completions": list})
}

func (h *Handler) GetCalendar(c *gin.Context) {
	userID, ok := h.requireWorkspace(c)
	if !ok {
		return
	}

	workspaceIDParam := c.Param("workspaceId")
	start, end, err := parseDateRange(c.Query("start"), c.Query("end"))
	if err != nil {
		h.responder.BadRequest(c, "Invalid start/end date")
		return
	}

	cal, err := h.service.GetCalendar(c.Request.Context(), userID, workspaceIDParam, start, end)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get calendar")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"days": cal.Days})
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	utc := parsed.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC), nil
}

func parseDateRange(startS, endS string) (start, end time.Time, err error) {
	if startS == "" || endS == "" {
		now := time.Now().UTC()
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		// Нормализуем end до начала дня в UTC
		end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, end, nil
	}
	start, err = time.Parse("2006-01-02", startS)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	startUTC := start.UTC()
	start = time.Date(startUTC.Year(), startUTC.Month(), startUTC.Day(), 0, 0, 0, 0, time.UTC)

	end, err = time.Parse("2006-01-02", endS)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endUTC := end.UTC()
	end = time.Date(endUTC.Year(), endUTC.Month(), endUTC.Day(), 0, 0, 0, 0, time.UTC)

	if end.Before(start) {
		start, end = end, start
	}
	return start, end, nil
}
