package scanner

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func TestScanDemoHTTP(t *testing.T) {
	root := repoRoot(t)

	graph, err := Scan([]string{"./demo/http"}, Options{Dir: root})
	if err != nil {
		t.Fatal(err)
	}

	if graph.Version != protocol.GraphVersion {
		t.Fatalf("version = %q, want %q", graph.Version, protocol.GraphVersion)
	}
	if graph.RootModule != "github.com/Maxim-Ba/debugviz" {
		t.Fatalf("root_module = %q, want github.com/Maxim-Ba/debugviz", graph.RootModule)
	}

	var packages, files, importEdges int
	packageIDs := make(map[string]struct{})
	for _, node := range graph.Nodes {
		switch node.Type {
		case protocol.NodeTypePackage:
			packages++
			packageIDs[node.ID] = struct{}{}
			if node.Path == "" {
				t.Fatalf("package node %q missing path", node.ID)
			}
		case protocol.NodeTypeFile:
			files++
			if node.Path == "" {
				t.Fatalf("file node %q missing path", node.ID)
			}
		}
	}
	if packages < 7 {
		t.Fatalf("packages = %d, want at least 7 demo/http packages", packages)
	}
	if files < 10 {
		t.Fatalf("files = %d, want at least 10 demo/http source files", files)
	}

	for _, edge := range graph.Edges {
		if edge.Type != protocol.EdgeTypeImports {
			continue
		}
		importEdges++
		if _, ok := packageIDs[edge.Source]; !ok {
			t.Fatalf("import edge source %q not in package nodes", edge.Source)
		}
		if _, ok := packageIDs[edge.Target]; !ok {
			t.Fatalf("import edge target %q not in package nodes", edge.Target)
		}
	}
	if importEdges == 0 {
		t.Fatal("expected import edges in demo/http graph")
	}

	wantEdge := importEdgeID(
		"github.com/Maxim-Ba/debugviz/demo/http/internal/handler",
		"github.com/Maxim-Ba/debugviz/demo/http/internal/service",
	)
	found := false
	for _, edge := range graph.Edges {
		if edge.ID == wantEdge {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing import edge %q (handler -> service)", wantEdge)
	}

	var entryPoints int
	entryRoutes := make(map[string]struct{})
	for _, node := range graph.Nodes {
		if node.Type != protocol.NodeTypeEntryPoint {
			continue
		}
		entryPoints++
		if node.Kind != protocol.EntryKindHTTP {
			t.Fatalf("entry_point %q: kind = %q, want http", node.ID, node.Kind)
		}
		method, _ := node.Metadata["method"].(string)
		path, _ := node.Metadata["path"].(string)
		entryRoutes[method+" "+path] = struct{}{}
	}
	if entryPoints < 6 {
		t.Fatalf("entry_points = %d, want at least 6 demo/http routes", entryPoints)
	}

	wantRoutes := []string{
		"GET /health",
		"GET /api/users/",
		"POST /api/users/",
		"GET /api/users/{id}",
		"GET /api/items/",
		"GET /api/items/{id}",
	}
	for _, route := range wantRoutes {
		if _, ok := entryRoutes[route]; !ok {
			t.Fatalf("missing HTTP route %q in graph", route)
		}
	}

	var entryHandles int
	for _, edge := range graph.Edges {
		if edge.Type == protocol.EdgeTypeEntryHandles {
			entryHandles++
		}
	}
	if entryHandles == 0 {
		t.Fatal("expected entry_handles edges for named handlers")
	}

	wantHandles := map[string]string{
		"GET /api/users/{id}": "func:demo/http/internal/handler/users.go:GetByID",
		"GET /api/items/{id}": "func:demo/http/internal/handler/items.go:GetByID",
	}
	for route, wantTarget := range wantHandles {
		method, path, _ := strings.Cut(route, " ")
		entryID := entryNodeID(protocol.EntryKindHTTP, method, path)
		found := false
		for _, edge := range graph.Edges {
			if edge.Type != protocol.EdgeTypeEntryHandles || edge.Source != entryID {
				continue
			}
			if edge.Target == wantTarget {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("route %q: missing entry_handles -> %q", route, wantTarget)
		}
	}
}

func TestScanDemoHTTPPerformance(t *testing.T) {
	root := repoRoot(t)

	start := time.Now()
	_, err := Scan([]string{"./demo/http/..."}, Options{Dir: root})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("scan took %v, want < 3s", elapsed)
	}
}
