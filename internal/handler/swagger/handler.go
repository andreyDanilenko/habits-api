package swagger

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "backend/docs"
)

func Register(r *gin.Engine, expose bool) {
	if !expose {
		return
	}
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/swagger/index.html") })
	r.GET("/favicon.ico", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
