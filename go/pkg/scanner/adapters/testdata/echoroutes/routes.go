package echoroutes

import "github.com/labstack/echo/v4"

func getUser(_ echo.Context) error { return nil }

func Register(e *echo.Echo) {
	e.GET("/users/:id", getUser)
	group := e.Group("/api")
	group.POST("/orders", getUser)
}
