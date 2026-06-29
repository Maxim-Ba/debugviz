package scanner_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func TestCallGraphDemoHTTP(t *testing.T) {
	root := repoRoot(t)
	graph, err := scanner.Scan([]string{"./demo/http"}, scanner.Options{Dir: root})
	if err != nil {
		t.Fatal(err)
	}

	wantCalls := []struct {
		source     string
		target     string
		confidence protocol.CallConfidence
	}{
		{
			source:     "func:demo/http/internal/handler/users.go:GetByID",
			target:     "func:demo/http/internal/service/user.go:GetByID",
			confidence: protocol.CallConfidenceStatic,
		},
		{
			source:     "func:demo/http/internal/service/user.go:GetByID",
			target:     "func:demo/http/internal/repository/user.go:FindByID",
			confidence: protocol.CallConfidenceStatic,
		},
	}

	for _, want := range wantCalls {
		found := false
		for _, edge := range graph.Edges {
			if edge.Type != protocol.EdgeTypeCalls {
				continue
			}
			if edge.Source == want.source && edge.Target == want.target {
				if edge.Confidence != want.confidence {
					t.Fatalf("call %s -> %s: confidence = %q, want %q", want.source, want.target, edge.Confidence, want.confidence)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing calls edge %s -> %s", want.source, want.target)
		}
	}

	entryID := "entry:http:GET:/api/users/{id}"
	files := entryReachableFiles(graph, entryID)
	if len(files) < 3 {
		t.Fatalf("entry %q reachable files = %d, want at least 3 (handler/service/repository)", entryID, len(files))
	}
}

func TestCallGraphInterfaceConfidence(t *testing.T) {
	root := repoRoot(t)
	graph, err := scanner.Scan(
		[]string{"./go/pkg/scanner/testdata/interfacecalls/..."},
		scanner.Options{Dir: root},
	)
	if err != nil {
		t.Fatal(err)
	}

	wantSource := "func:go/pkg/scanner/testdata/interfacecalls/calls.go:Get"
	wantTarget := "func:go/pkg/scanner/testdata/interfacecalls/calls.go:FindByID"

	found := false
	for _, edge := range graph.Edges {
		if edge.Type != protocol.EdgeTypeCalls {
			continue
		}
		if edge.Source != wantSource || edge.Target != wantTarget {
			continue
		}
		if edge.Confidence != protocol.CallConfidenceInterface {
			t.Fatalf("interface call confidence = %q, want interface", edge.Confidence)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("missing interface call edge %s -> %s", wantSource, wantTarget)
	}
}

func entryReachableFiles(g *protocol.Graph, entryID string) map[string]struct{} {
	adj := make(map[string][]string)
	for _, edge := range g.Edges {
		switch edge.Type {
		case protocol.EdgeTypeEntryHandles, protocol.EdgeTypeCalls:
			adj[edge.Source] = append(adj[edge.Source], edge.Target)
		}
	}

	files := make(map[string]struct{})
	queue := []string{entryID}
	seen := map[string]struct{}{entryID: {}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if file := funcFileFromNodeID(cur); file != "" {
			files[file] = struct{}{}
		}
		for _, next := range adj[cur] {
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			queue = append(queue, next)
		}
	}
	return files
}

func funcFileFromNodeID(nodeID string) string {
	const prefix = "func:"
	if len(nodeID) <= len(prefix) {
		return ""
	}
	if nodeID[:len(prefix)] != prefix {
		return ""
	}
	rest := nodeID[len(prefix):]
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == ':' {
			return rest[:i]
		}
	}
	return ""
}
