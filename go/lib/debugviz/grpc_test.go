package debugviz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryServerInterceptorRootSpan(t *testing.T) {
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

	interceptor := UnaryServerInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		if SpanFromContext(ctx) == nil {
			t.Fatal("expected span in context")
		}
		return "ok", nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-trace-id", "grpc-trace-1"))
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}, handler)
	if err != nil {
		t.Fatal(err)
	}

	waitForSpans(t, &mu, &received, 1)
	span := received[0]
	if span.TraceID != "grpc-trace-1" {
		t.Fatalf("trace_id = %q, want grpc-trace-1", span.TraceID)
	}
	if span.EntryKind != protocol.EntryKindGRPC {
		t.Fatalf("entry_kind = %q, want grpc", span.EntryKind)
	}
	if span.EntryName != "user.v1.UserService/GetUser" {
		t.Fatalf("entry_name = %q", span.EntryName)
	}
}

func TestConfigureFromEnv(t *testing.T) {
	resetRuntime(t)
	t.Setenv("DEBUGVIZ_ENABLED", "true")
	t.Setenv("DEBUGVIZ_SERVER_URL", "http://127.0.0.1:9")
	t.Setenv("DEBUGVIZ_BATCH_SIZE", "10")

	if err := ConfigureFromEnv(); err != nil {
		t.Fatal(err)
	}
	if !isEnabled() {
		t.Fatal("expected enabled from env")
	}
	globalMu.RLock()
	cfg := globalCfg
	globalMu.RUnlock()
	if cfg.BatchSize != 10 {
		t.Fatalf("batch size = %d, want 10", cfg.BatchSize)
	}
}

func TestConfigureFromYAML(t *testing.T) {
	resetRuntime(t)
	path := filepath.Join(t.TempDir(), "debugviz.yaml")
	content := "server_url: http://yaml.example:4000\nservice_name: yaml-app\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEBUGVIZ_ENABLED", "true")
	t.Setenv("DEBUGVIZ_SERVER_URL", "http://env.example:4000")

	if err := ConfigureFromYAML(path); err != nil {
		t.Fatal(err)
	}
	globalMu.RLock()
	cfg := globalCfg
	globalMu.RUnlock()
	if cfg.ServerURL != "http://env.example:4000" {
		t.Fatalf("server url = %q, want env override", cfg.ServerURL)
	}
	if cfg.ServiceName != "yaml-app" {
		t.Fatalf("service name = %q, want yaml-app", cfg.ServiceName)
	}
}

func TestRunJobRootSpan(t *testing.T) {
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

	err := RunJob(context.Background(), "ProcessOrder", func(ctx context.Context) error {
		if SpanFromContext(ctx) == nil {
			t.Fatal("expected span in job context")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	waitForSpans(t, &mu, &received, 1)
	if received[0].EntryKind != protocol.EntryKindWorker {
		t.Fatalf("entry_kind = %q, want worker", received[0].EntryKind)
	}
	if received[0].EntryName != "ProcessOrder" {
		t.Fatalf("entry_name = %q", received[0].EntryName)
	}
}
