package debugviz

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func TestGinMiddlewareRootSpan(t *testing.T) {
	resetRuntime(t)
	gin.SetMode(gin.TestMode)

	var mu sync.Mutex
	var received []protocol.TraceEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		var payload struct {
			Spans []protocol.TraceEvent `json:"spans"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		mu.Lock()
		received = append(received, payload.Spans...)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	if err := Configure(Config{ServerURL: srv.URL, Enabled: true, BatchSize: 1}); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(GinMiddleware(HTTPMiddlewareConfig{ServiceName: "demo-gin"}))
	r.GET("/api/users/:id", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	req.Header.Set(traceIDHeader, "gin-trace-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get(traceIDHeader); got != "gin-trace-1" {
		t.Fatalf("response trace header = %q, want gin-trace-1", got)
	}

	waitForSpans(t, &mu, &received, 1)

	mu.Lock()
	defer mu.Unlock()
	span := received[0]
	if span.TraceID != "gin-trace-1" {
		t.Fatalf("trace_id = %q, want gin-trace-1", span.TraceID)
	}
	if span.EntryKind != protocol.EntryKindHTTP {
		t.Fatalf("entry_kind = %q, want http", span.EntryKind)
	}
	if span.EntryName != "GET /api/users/1" {
		t.Fatalf("entry_name = %q, want GET /api/users/1", span.EntryName)
	}
}

func TestGinMiddlewareErrorStatus(t *testing.T) {
	resetRuntime(t)
	gin.SetMode(gin.TestMode)

	var mu sync.Mutex
	var received []protocol.TraceEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		var payload struct {
			Spans []protocol.TraceEvent `json:"spans"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		mu.Lock()
		received = append(received, payload.Spans...)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	if err := Configure(Config{ServerURL: srv.URL, Enabled: true, BatchSize: 1}); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(GinMiddleware(HTTPMiddlewareConfig{}))
	r.GET("/missing", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	waitForSpans(t, &mu, &received, 1)
	mu.Lock()
	defer mu.Unlock()
	if received[0].Status != protocol.SpanStatusError {
		t.Fatalf("status = %q, want error", received[0].Status)
	}
}
