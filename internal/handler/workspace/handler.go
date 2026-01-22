package workspace

import (
	"backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   *workspace.Service
	validate  *validator.Validate
	responder *response.Responder
}

func NewHandler(
	service *workspace.Service,
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
	r.GET(RouteMembers, h.GetMembers)
	r.POST(RouteSwitch, h.Switch)
	r.GET(RouteModules, h.GetModules)
}

func (h *Handler) List(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "list workspaces"})
}

func (h *Handler) Create(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "create workspace"})
}

func (h *Handler) Get(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get workspace"})
}

func (h *Handler) Update(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "update workspace"})
}

func (h *Handler) Delete(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "delete workspace"})
}

func (h *Handler) GetMembers(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get members"})
}

func (h *Handler) Switch(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "switch workspace"})
}

func (h *Handler) GetModules(c *gin.Context) {
	// TODO: implement
	c.JSON(200, gin.H{"message": "get modules"})
}
