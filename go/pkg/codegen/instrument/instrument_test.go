package instrument

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstrumentGoldenWithContext(t *testing.T) {
	runGoldenTest(t, "with_context.input.go", "with_context.golden.go", Config{
		Include: []string{"**"},
	})
}

func TestInstrumentGoldenBlankContextParam(t *testing.T) {
	runGoldenTest(t, "blank_context.input.go", "blank_context.golden.go", Config{
		Include: []string{"**"},
	})
}

func TestInstrumentGoldenNoContext(t *testing.T) {
	runGoldenTest(t, "no_context.input.go", "no_context.golden.go", Config{
		Include:        []string{"**"},
		AllowNoContext: true,
	})
}

func TestInstrumentGoldenIdempotent(t *testing.T) {
	inputPath := filepath.Join("testdata", "with_context.input.go")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Include: []string{"**"}}
	pkgName := packageNameFromSource(string(input))

	first, count, err := InstrumentFile(input, inputPath, cfg, pkgName)
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected first pass to instrument functions")
	}

	second, againCount, err := InstrumentFile(first, inputPath, cfg, pkgName)
	if err != nil {
		t.Fatal(err)
	}
	if againCount != 0 {
		t.Fatalf("idempotent re-run instrumented %d functions", againCount)
	}
	if normalizeGo(second) != normalizeGo(first) {
		t.Fatal("idempotent re-run changed output")
	}
}

func runGoldenTest(t *testing.T, inputName, goldenName string, cfg Config) {
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

	pkgName := packageNameFromSource(string(input))
	got, count, err := InstrumentFile(input, inputPath, cfg, pkgName)
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected at least one instrumented function")
	}
	if normalizeGo(got) != normalizeGo(golden) {
		t.Fatalf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, golden)
	}

	again, againCount, err := InstrumentFile(got, inputPath, cfg, pkgName)
	if err != nil {
		t.Fatal(err)
	}
	if againCount != 0 {
		t.Fatalf("idempotent re-run instrumented %d functions", againCount)
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

func packageNameFromSource(src string) string {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "package "))
		}
	}
	return "main"
}

func TestPathMatches(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"demo/http/internal/service/**", "demo/http/internal/service/item.go", true},
		{"demo/http/internal/service/**", "demo/http/internal/handler/items.go", false},
		{"**/*_test.go", "demo/http/main_test.go", true},
		{"**/mocks/**", "internal/mocks/repo.go", true},
		{"internal/repository/**", "demo/http/internal/repository/item.go", true},
	}
	for _, tc := range cases {
		if got := pathMatches(tc.path, tc.pattern); got != tc.want {
			t.Fatalf("pathMatches(%q, %q) = %v, want %v", tc.path, tc.pattern, got, tc.want)
		}
	}
}

func TestShouldInstrumentFile(t *testing.T) {
	cfg := Config{
		Include: []string{"demo/http/internal/service/**", "demo/http/internal/repository/**"},
		Exclude: []string{"**/*_test.go"},
	}
	if !shouldInstrumentFile("demo/http/internal/service/item.go", cfg, false) {
		t.Fatal("expected service file to match include")
	}
	if shouldInstrumentFile("demo/http/internal/handler/items.go", cfg, false) {
		t.Fatal("handler file should not match include-only config")
	}
	if shouldInstrumentFile("demo/http/internal/service/item_test.go", cfg, false) {
		t.Fatal("test file should be excluded")
	}
}

func TestFuncHasSkipDirective(t *testing.T) {
	const src = `package p

import "context"

//debugviz:instrument skip
func Skipped(ctx context.Context) error {
	return nil
}

func Included(ctx context.Context) error {
	return nil
}
`
	fset, file, err := parseSource("skip.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	pkg, info := typeCheckFile(fset, file, "p")
	candidates := analyzeFile(file, pkg, info, Config{Include: []string{"**"}})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].Decl.Name.Name != "Included" {
		t.Fatalf("instrumented %s, want Included", candidates[0].Decl.Name.Name)
	}
}

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debugviz.yaml")
	content := `
include:
  - internal/service/**
allow_no_context: true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Include) != 1 {
		t.Fatalf("include = %#v", cfg.Include)
	}
	if !cfg.AllowNoContext {
		t.Fatal("expected allow_no_context")
	}
	if !cfg.requireContext() {
		t.Fatal("require_context should default to true")
	}
}
