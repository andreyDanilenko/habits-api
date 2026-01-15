package habits

import (
	"backend/internal/service/habits"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *habits.Service
}

func NewHandler(service *habits.Service) *Handler {
	return &Handler{
		service: service,
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

func (h *Handler) List(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "list habits"})
}

func (h *Handler) Create(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "create habit"})
}

func (h *Handler) Get(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get habit"})
}

func (h *Handler) Update(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "update habit"})
}

func (h *Handler) Delete(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "delete habit"})
}

func (h *Handler) Complete(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "complete habit"})
}

func (h *Handler) Toggle(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "toggle habit"})
}

func (h *Handler) GetStats(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get stats"})
}

func (h *Handler) GetCompletions(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get completions"})
}

func (h *Handler) GetCalendar(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get calendar"})
}
