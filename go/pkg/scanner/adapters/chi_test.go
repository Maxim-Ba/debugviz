package adapters_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func loadDemoHTTPPackages(t *testing.T) []*packages.Package {
	t.Helper()
	root := repoRoot(t)
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedImports | packages.NeedModule | packages.NeedCompiledGoFiles,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, "./demo/http/...")
	if err != nil {
		t.Fatal(err)
	}
	return pkgs
}

func TestChiDiscoverDemoHTTPRoutes(t *testing.T) {
	pkgs := loadDemoHTTPPackages(t)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewChi().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected chi routes in demo/http")
	}

	routes := make(map[string]struct{})
	for _, entry := range entries {
		routes[entry.Method+" "+entry.Path] = struct{}{}
	}

	want := []string{
		"GET /health",
		"GET /api/users/",
		"POST /api/users/",
		"GET /api/users/{id}",
		"GET /api/items/",
		"GET /api/items/{id}",
	}
	for _, route := range want {
		if _, ok := routes[route]; !ok {
			t.Fatalf("missing route %q, got %v", route, routes)
		}
	}
}
