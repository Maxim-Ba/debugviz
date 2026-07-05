package wire

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWireGoldenStdlibHTTP(t *testing.T) {
	runGoldenTest(t, "stdlib_http.input.go", "stdlib_http.golden.go", Config{
		ServiceName: "my-app",
		Wire: WireConfig{
			HTTP: HTTPWireConfig{ListenAndServe: true},
		},
	}, true)
}

func TestWireGoldenChiRouter(t *testing.T) {
	runGoldenTest(t, "chi_router.input.go", "chi_router.golden.go", Config{
		ServiceName: "demo-http",
		Wire: WireConfig{
			HTTP: HTTPWireConfig{},
		},
	}, false)
}

func TestWireGoldenGRPCMain(t *testing.T) {
	runGoldenTest(t, "grpc_main.input.go", "grpc_main.golden.go", Config{
		ServiceName: "demo-grpc",
		Wire: WireConfig{
			GRPC: GRPCWireConfig{NewServer: true},
		},
	}, true)
}

func TestWireGoldenCLICobra(t *testing.T) {
	runGoldenTest(t, "cli_cobra.input.go", "cli_cobra.golden.go", Config{
		ServiceName: "demo-cli",
		Wire: WireConfig{
			CLI: CLIWireConfig{CobraExecute: true},
		},
	}, true)
}

func TestWireGoldenWorkerMain(t *testing.T) {
	runGoldenTest(t, "worker_main.input.go", "worker_main.golden.go", Config{
		ServiceName: "demo-worker",
		Wire: WireConfig{
			Worker: WorkerWireConfig{
				Targets: []WorkerJobTarget{{Name: "OrderConsumer.Process"}},
			},
		},
	}, true)
}

func TestWireGoldenAppAnnotation(t *testing.T) {
	runGoldenTest(t, "app_annotation.input.go", "app_annotation.golden.go", Config{
		Wire: WireConfig{
			HTTP: HTTPWireConfig{ListenAndServe: true},
		},
	}, true)
}

func TestWireGoldenConfigureOnly(t *testing.T) {
	runGoldenTest(t, "configure_only.input.go", "configure_only.golden.go", Config{}, true)
}

func TestWireGoldenGRPCExistingOpts(t *testing.T) {
	runGoldenTest(t, "grpc_existing_opts.input.go", "grpc_existing_opts.golden.go", Config{
		Wire: WireConfig{GRPC: GRPCWireConfig{NewServer: true}},
	}, true)
}

func TestWireGoldenHTTPSkip(t *testing.T) {
	runGoldenTest(t, "http_skip.input.go", "http_skip.golden.go", Config{
		Wire: WireConfig{HTTP: HTTPWireConfig{ListenAndServe: true}},
	}, true)
}

func TestWireGoldenWireSkip(t *testing.T) {
	inputPath := filepath.Join("testdata", "wire_skip.input.go")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	got, plan, err := WireFile(input, inputPath, Config{
		Wire: WireConfig{HTTP: HTTPWireConfig{ListenAndServe: true}},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if planHasChanges(plan) {
		t.Fatalf("expected no changes for wire skip, plan=%v", plan)
	}
	if normalizeGo(got) != normalizeGo(input) {
		t.Fatal("wire skip changed file")
	}
}

func TestWireGoldenIdempotent(t *testing.T) {
	inputPath := filepath.Join("testdata", "stdlib_http.input.go")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		ServiceName: "demo-http",
		Wire:        WireConfig{HTTP: HTTPWireConfig{ListenAndServe: true}},
	}
	first, plan, err := WireFile(input, inputPath, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if !planHasChanges(plan) {
		t.Fatal("expected first pass to wire main")
	}
	second, again, err := WireFile(first, inputPath, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if planHasChanges(again) {
		t.Fatal("idempotent re-run changed plan")
	}
	if normalizeGo(second) != normalizeGo(first) {
		t.Fatal("idempotent re-run changed output")
	}
}

func TestParseAppAnnotation(t *testing.T) {
	const src = `//debugviz:app name=my-app server=http://localhost:4000
package main
`
	ann, err := parseAppAnnotationFromSource([]byte(src), "main.go")
	if err != nil {
		t.Fatal(err)
	}
	if ann.Name != "my-app" || ann.Server != "http://localhost:4000" {
		t.Fatalf("annotation = %#v", ann)
	}
}

func TestParseAppAnnotationInvalid(t *testing.T) {
	const src = `//debugviz:app badtoken
package main
`
	_, err := parseAppAnnotationFromSource([]byte(src), "main.go")
	if err == nil {
		t.Fatal("expected error for invalid annotation")
	}
	if !strings.Contains(err.Error(), "badtoken") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadConfigWireSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debugviz.yaml")
	content := `
service_name: demo-http
wire:
  main: demo/http/main.go
  http:
    listen_and_serve: true
  grpc:
    new_server: true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServiceName != "demo-http" {
		t.Fatalf("service_name = %q", cfg.ServiceName)
	}
	if cfg.Wire.Main != "demo/http/main.go" {
		t.Fatalf("wire.main = %q", cfg.Wire.Main)
	}
	if !cfg.Wire.HTTP.ListenAndServe {
		t.Fatal("expected listen_and_serve")
	}
	if !cfg.Wire.GRPC.NewServer {
		t.Fatal("expected grpc new_server")
	}
}

func runGoldenTest(t *testing.T, inputName, goldenName string, cfg Config, isMain bool) {
	t.Helper()

	inputPath := filepath.Join("testdata", inputName)
	goldenPath := filepath.Join("testdata", goldenName)

	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}

	got, plan, err := WireFile(input, inputPath, cfg, isMain)
	if err != nil {
		t.Fatal(err)
	}
	if !planHasChanges(plan) {
		t.Fatal("expected wire changes")
	}
	if normalizeGo(got) != normalizeGo(golden) {
		t.Fatalf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, golden)
	}

	again, againPlan, err := WireFile(got, inputPath, cfg, isMain)
	if err != nil {
		t.Fatal(err)
	}
	if planHasChanges(againPlan) {
		t.Fatalf("idempotent re-run changed plan: %v", againPlan)
	}
	if normalizeGo(again) != normalizeGo(golden) {
		t.Fatal("idempotent re-run changed output")
	}
}

func normalizeGo(src []byte) string {
	formatted, err := format.Source(src)
	if err != nil {
		return string(src)
	}
	return string(formatted)
}
