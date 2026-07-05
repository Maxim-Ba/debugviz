package debugviz

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// EchoMiddleware returns echo-compatible middleware wrapping each request with a root span.
func EchoMiddleware(cfg HTTPMiddlewareConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.SetRequest(r)
				err := next(c)
				if span := SpanFromContext(r.Context()); span != nil {
					if err != nil {
						var he *echo.HTTPError
						if errors.As(err, &he) && he.Code >= http.StatusBadRequest {
							span.Status = protocol.SpanStatusError
						}
					} else if c.Response().Status >= http.StatusBadRequest {
						span.Status = protocol.SpanStatusError
					}
				}
			}), cfg)
			handler.ServeHTTP(c.Response().Writer, c.Request())
			return nil
		}
	}
}
