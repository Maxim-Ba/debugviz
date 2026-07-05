package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func New(serviceName string) *echo.Echo {
	e := echo.New()
	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	return e
}
