package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func New(serviceName string) *gin.Engine {
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}
