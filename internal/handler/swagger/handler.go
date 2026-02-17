package swagger

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "backend/docs"
)

func Register(r *gin.Engine, expose bool, user, password string) {
	if !expose {
		return
	}
	useBasicAuth := user != "" && password != ""

	var group *gin.RouterGroup
	if useBasicAuth {
		group = r.Group("/", gin.BasicAuth(gin.Accounts{user: password}))
	} else {
		group = r.Group("/")
	}

	group.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/swagger/index.html") })
	r.GET("/favicon.ico", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	if useBasicAuth {
		r.Group("/swagger").Use(gin.BasicAuth(gin.Accounts{user: password})).GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}
