package ginroutes

import "github.com/gin-gonic/gin"

func getUser(_ *gin.Context) {}
func createUser(_ *gin.Context) {}

func Register(r *gin.Engine) {
	r.GET("/users/:id", getUser)
	api := r.Group("/api")
	api.POST("/users", createUser)
}
