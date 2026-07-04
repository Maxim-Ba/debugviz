package debugviz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func resetBench(b *testing.B) {
	b.Helper()
	globalMu.Lock()
	globalOn = false
	globalExport = nil
	globalCfg = Config{}
	sourceRoot = ""
	globalMu.Unlock()
}

func benchExportServer(b *testing.B) *httptest.Server {
	b.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
}

func benchRequest() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
}

// BenchmarkHandlerBaseline measures a plain handler with tracing disabled.
func BenchmarkHandlerBaseline(b *testing.B) {
	resetBench(b)
	if err := Configure(Config{Enabled: false}); err != nil {
		b.Fatal(err)
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := benchRequest()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		inner.ServeHTTP(rec, req)
	}
}

// BenchmarkHTTPMiddleware measures root span middleware overhead per request.
func BenchmarkHTTPMiddleware(b *testing.B) {
	resetBench(b)
	srv := benchExportServer(b)
	defer srv.Close()

	if err := Configure(Config{
		ServerURL: srv.URL,
		Enabled:   true,
		BatchSize: 256,
	}); err != nil {
		b.Fatal(err)
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := HTTPMiddleware(inner, HTTPMiddlewareConfig{ServiceName: "bench"})

	req := benchRequest()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkManualSpans measures one inner StartSpan per request (manual instrumentation).
func BenchmarkManualSpans(b *testing.B) {
	resetBench(b)
	srv := benchExportServer(b)
	defer srv.Close()

	if err := Configure(Config{
		ServerURL: srv.URL,
		Enabled:   true,
		BatchSize: 256,
	}); err != nil {
		b.Fatal(err)
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, end := StartSpan(r.Context(), "handler.Work")
		defer end()
		_ = ctx
		w.WriteHeader(http.StatusOK)
	})
	handler := HTTPMiddleware(inner, HTTPMiddlewareConfig{ServiceName: "bench"})

	req := benchRequest()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkCodegenSpans simulates a short instrumented call chain (handler + service + repo).
func BenchmarkCodegenSpans(b *testing.B) {
	resetBench(b)
	srv := benchExportServer(b)
	defer srv.Close()

	if err := Configure(Config{
		ServerURL: srv.URL,
		Enabled:   true,
		BatchSize: 256,
	}); err != nil {
		b.Fatal(err)
	}

	work := func(ctx context.Context) {
		ctx, endRepo := StartSpan(ctx, "repository.FindByID")
		endRepo()

		ctx, endSvc := StartSpan(ctx, "service.GetByID")
		endSvc()

		ctx, endH := StartSpan(ctx, "handler.GetByID")
		endH()
		_ = ctx
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		work(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := HTTPMiddleware(inner, HTTPMiddlewareConfig{ServiceName: "bench"})

	req := benchRequest()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
