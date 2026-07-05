package debugviz

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func resetRuntime(t *testing.T) {
	t.Helper()
	globalMu.Lock()
	globalOn = false
	globalExport = nil
	globalCfg = Config{}
	sourceRoot = ""
	globalMu.Unlock()
}

func TestStartSpanDisabledIsNoOp(t *testing.T) {
	resetRuntime(t)

	ctx := context.Background()
	ctx, end := StartSpan(ctx, "demo")
	end()

	if got := SpanFromContext(ctx); got != nil {
		t.Fatalf("expected nil span when disabled, got %#v", got)
	}
}

func TestStartSpanExportsEvent(t *testing.T) {
	resetRuntime(t)

	var mu sync.Mutex
	var received []protocol.TraceEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		var payload struct {
			Spans []protocol.TraceEvent `json:"spans"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, payload.Spans...)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	if err := Configure(Config{
		ServerURL: srv.URL,
		Enabled:   true,
		BatchSize: 1,
	}); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, end := StartSpan(ctx, "service.DoWork")
	end()

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received %d spans, want 1", len(received))
	}
	if received[0].Name != "service.DoWork" {
		t.Fatalf("name = %q, want service.DoWork", received[0].Name)
	}
	if received[0].Status != protocol.SpanStatusOK {
		t.Fatalf("status = %q, want ok", received[0].Status)
	}
}

func TestHTTPMiddlewareRootSpan(t *testing.T) {
	resetRuntime(t)

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

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	handler := HTTPMiddleware(inner, HTTPMiddlewareConfig{ServiceName: "demo-http"})

	req := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	req.Header.Set(traceIDHeader, "trace-demo-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(traceIDHeader); got != "trace-demo-1" {
		t.Fatalf("response trace header = %q, want trace-demo-1", got)
	}

	waitForSpans(t, &mu, &received, 1)

	mu.Lock()
	defer mu.Unlock()
	span := received[0]
	if span.TraceID != "trace-demo-1" {
		t.Fatalf("trace_id = %q, want trace-demo-1", span.TraceID)
	}
	if span.EntryKind != protocol.EntryKindHTTP {
		t.Fatalf("entry_kind = %q, want http", span.EntryKind)
	}
	if span.EntryName != "GET /api/users/1" {
		t.Fatalf("entry_name = %q, want GET /api/users/1", span.EntryName)
	}
}

func TestExporterRingBufferDropsOldest(t *testing.T) {
	ring := newSpanRing(2)
	ring.push(protocol.TraceEvent{SpanID: "a"})
	ring.push(protocol.TraceEvent{SpanID: "b"})
	ring.push(protocol.TraceEvent{SpanID: "c"})

	first, ok := ring.pop()
	if !ok || first.SpanID != "b" {
		t.Fatalf("first pop = %#v, want span b", first)
	}
	second, ok := ring.pop()
	if !ok || second.SpanID != "c" {
		t.Fatalf("second pop = %#v, want span c", second)
	}
}

func waitForSpans(t *testing.T, mu *sync.Mutex, received *[]protocol.TraceEvent, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(*received)
		mu.Unlock()
		if n >= want || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*received) < want {
		t.Fatalf("received %d spans, want >= %d", len(*received), want)
	}
}
