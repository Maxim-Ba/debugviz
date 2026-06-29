package scanner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

const demoHTTPGoldenFile = "testdata/demo_http.golden.json"

func TestScanDemoHTTPGolden(t *testing.T) {
	root := repoRoot(t)

	graph, err := Scan([]string{"./demo/http"}, Options{
		Dir:       root,
		Framework: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := marshalGraphForGolden(graph)
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join(filepath.Dir(testFile(t)), demoHTTPGoldenFile)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated golden file %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v (run with UPDATE_GOLDEN=1 to create)", err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("graph differs from %s (run with UPDATE_GOLDEN=1 to refresh)", demoHTTPGoldenFile)
	}
}

func marshalGraphForGolden(graph *protocol.Graph) ([]byte, error) {
	copy := *graph
	copy.GeneratedAt = ""
	data, err := json.MarshalIndent(&copy, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func testFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return file
}
