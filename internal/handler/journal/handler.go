package journal

import (
	"backend/internal/service/journal"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *journal.Service
}

func NewHandler(service *journal.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET(RouteList, h.List)
	r.POST(RouteCreate, h.Create)
	r.GET(RouteGet, h.GetByDate)
	r.PUT(RouteUpdate, h.Update)
	r.DELETE(RouteDelete, h.Delete)
}

func (h *Handler) List(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "list journals"})
}

func (h *Handler) Create(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "create journal"})
}

func (h *Handler) GetByDate(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get journal by date"})
}

func (h *Handler) Update(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "update journal"})
}

func (h *Handler) Delete(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "delete journal"})
}
