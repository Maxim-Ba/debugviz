package router

import (
	"net/http"

	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"github.com/gin-gonic/gin"
)

func New(serviceName string) *gin.Engine {
	r := gin.New()
	r.Use(debugviz.GinMiddleware(debugviz.HTTPMiddlewareConfig{ServiceName: serviceName}))
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}
