package debugviz

import (
	"context"
	"net/http"
	"strings"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

const traceIDHeader = "X-Trace-ID"

type traceIDContextKey struct{}

// HTTPMiddleware wraps an http.Handler with a root span per request.
func HTTPMiddleware(next http.Handler, cfg HTTPMiddlewareConfig) http.Handler {
	if next == nil {
		panic("debugviz: HTTPMiddleware requires non-nil handler")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		traceID := strings.TrimSpace(r.Header.Get(traceIDHeader))
		if traceID == "" {
			traceID = newTraceID()
		}

		entryName := r.Method + " " + r.URL.Path
		ctx := withTraceID(r.Context(), traceID)
		ctx, end := startRootSpan(ctx, protocol.EntryKindHTTP, entryName)
		defer end()

		span := SpanFromContext(ctx)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		w.Header().Set(traceIDHeader, traceID)

		next.ServeHTTP(rec, r.WithContext(ctx))

		if span != nil {
			if rec.status >= http.StatusBadRequest {
				span.Status = protocol.SpanStatusError
			}
			span.Name = entryName
			if cfg.ServiceName != "" {
				span.Name = cfg.ServiceName + " " + entryName
			}
		}
	})
}

// ChiMiddleware returns chi-compatible middleware wrapping each request with a root span.
func ChiMiddleware(cfg HTTPMiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return HTTPMiddleware(next, cfg)
	}
}

func withTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDContextKey{}, traceID)
}

func traceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(traceIDContextKey{}).(string)
	return id
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
