package scanner

import (
	"path/filepath"
	"runtime"
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
