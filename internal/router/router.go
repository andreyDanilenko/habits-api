package router

import (
	"backend/internal/middleware"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine *gin.Engine
}

func New(responder *response.Responder) *Router {
	engine := gin.Default()
	engine.Use(middleware.ErrorHandler(responder))

	return &Router{engine: engine}
}

func (r *Router) Handler() *gin.Engine {
	return r.engine
}

func (r *Router) GET(path string, handlers ...gin.HandlerFunc) {
	r.engine.GET(path, handlers...)
}

func (r *Router) POST(path string, handlers ...gin.HandlerFunc) {
	r.engine.POST(path, handlers...)
}

func (r *Router) PUT(path string, handlers ...gin.HandlerFunc) {
	r.engine.PUT(path, handlers...)
}

func (r *Router) DELETE(path string, handlers ...gin.HandlerFunc) {
	r.engine.DELETE(path, handlers...)
}

func (r *Router) Group(relativePath string) *gin.RouterGroup {
	return r.engine.Group(relativePath)
}
