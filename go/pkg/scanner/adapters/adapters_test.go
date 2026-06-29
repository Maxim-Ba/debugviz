package adapters_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters"
)

func loadPackages(t *testing.T, pattern string) []*packages.Package {
	t.Helper()
	root := repoRoot(t)
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedModule | packages.NeedCompiledGoFiles,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		t.Fatal(err)
	}
	return pkgs
}

func TestGinDiscoverRoutes(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/ginroutes/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewGin().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	routes := routeSet(entries)
	for _, want := range []string{"GET /users/:id", "POST /api/users"} {
		if _, ok := routes[want]; !ok {
			t.Fatalf("missing route %q, got %v", want, routes)
		}
	}
}

func TestEchoDiscoverRoutes(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/echoroutes/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewEcho().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	routes := routeSet(entries)
	for _, want := range []string{"GET /users/:id", "POST /api/orders"} {
		if _, ok := routes[want]; !ok {
			t.Fatalf("missing route %q, got %v", want, routes)
		}
	}
}

func TestStdlibDiscoverRoutes(t *testing.T) {
	pattern := "./go/pkg/scanner/adapters/testdata/stdlibroutes/..."
	pkgs := loadPackages(t, pattern)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.NewStdlib().Discover(ctx, pkgs)
	if err != nil {
		t.Fatal(err)
	}

	routes := routeSet(entries)
	for _, want := range []string{"GET /health", "GET /api/items/"} {
		if _, ok := routes[want]; !ok {
			t.Fatalf("missing route %q, got %v", want, routes)
		}
	}
}

func TestManualEntriesFromYAML(t *testing.T) {
	root := repoRoot(t)
	configDir := filepath.Join(root, "go", "pkg", "scanner", "adapters", "testdata", "manual")
	pkgs := loadDemoHTTPPackages(t)
	ctx := adapters.NewScanContext("github.com/Maxim-Ba/debugviz", pkgs)

	entries, err := adapters.LoadManualEntries(configDir, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Method != "GET" || entries[0].Path != "/api/custom" {
		t.Fatalf("unexpected manual entry: %+v", entries[0])
	}
}

func routeSet(entries []adapters.EntryPoint) map[string]struct{} {
	routes := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		routes[entry.Method+" "+entry.Path] = struct{}{}
	}
	return routes
}
