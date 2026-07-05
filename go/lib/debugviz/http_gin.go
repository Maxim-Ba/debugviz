package debugviz

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// GinMiddleware returns gin-compatible middleware wrapping each request with a root span.
func GinMiddleware(cfg HTTPMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
			if span := SpanFromContext(r.Context()); span != nil && c.Writer.Status() >= http.StatusBadRequest {
				span.Status = protocol.SpanStatusError
			}
		}), cfg)
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
