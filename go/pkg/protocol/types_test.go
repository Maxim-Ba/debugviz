package protocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func exampleGraphPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "schemas", "examples", "demo-http-graph.json")
}

func TestDemoHTTPGraphUnmarshal(t *testing.T) {
	data, err := os.ReadFile(exampleGraphPath(t))
	if err != nil {
		t.Fatal(err)
	}

	var graph Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		t.Fatalf("unmarshal graph: %v", err)
	}

	if graph.Version != GraphVersion {
		t.Fatalf("version = %q, want %q", graph.Version, GraphVersion)
	}
	if len(graph.Nodes) == 0 {
		t.Fatal("expected nodes in demo graph")
	}
	if len(graph.Edges) == 0 {
		t.Fatal("expected edges in demo graph")
	}

	var entryCount int
	for _, node := range graph.Nodes {
		if node.Type == NodeTypeEntryPoint {
			entryCount++
			if node.Kind != EntryKindHTTP {
				t.Fatalf("entry_point %q: kind = %q, want http", node.ID, node.Kind)
			}
			if node.Metadata == nil {
				t.Fatalf("entry_point %q: metadata is nil", node.ID)
			}
		}
	}
	if entryCount != 2 {
		t.Fatalf("entry_points = %d, want 2", entryCount)
	}

	meta := ComputeGraphMeta(&graph)
	if meta.EntryPoints != 2 {
		t.Fatalf("meta.entry_points = %d, want 2", meta.EntryPoints)
	}
	if meta.Nodes != len(graph.Nodes) {
		t.Fatalf("meta.nodes = %d, want %d", meta.Nodes, len(graph.Nodes))
	}
}

func TestMiddlewareChainEdgeMarshalsOrderZero(t *testing.T) {
	order := 0
	edge := Edge{
		Type:   EdgeTypeMiddlewareChain,
		Source: "entry:http:GET:/",
		Target: "mw:demo/http/internal/middleware/logging.go:Logging",
		Order:  &order,
	}

	data, err := json.Marshal(edge)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("invalid json: %s", data)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	got, ok := raw["order"]
	if !ok {
		t.Fatalf("order missing from json: %s", data)
	}
	if got != float64(0) {
		t.Fatalf("order = %v, want 0", got)
	}
}

func TestTraceEventRoundTrip(t *testing.T) {
	parentID := "parent-span-1"
	event := TraceEvent{
		TraceID:      "trace-1",
		SpanID:       "span-root",
		ParentSpanID: nil,
		Name:         "UserHandler.GetByID",
		File:         "demo/http/internal/handler/users.go",
		Line:         18,
		StartUs:      1710000000,
		DurationUs:   1200,
		Status:       SpanStatusOK,
		EntryKind:    EntryKindHTTP,
		EntryName:    "GET /api/users/1",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}

	var decoded TraceEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.EntryKind != EntryKindHTTP {
		t.Fatalf("entry_kind = %q, want http", decoded.EntryKind)
	}
	if decoded.EntryName != event.EntryName {
		t.Fatalf("entry_name = %q, want %q", decoded.EntryName, event.EntryName)
	}

	child := TraceEvent{
		TraceID:      event.TraceID,
		SpanID:       "span-child",
		ParentSpanID: &parentID,
		Name:         "UserService.GetByID",
		File:         "demo/http/internal/service/user.go",
		Line:         24,
		StartUs:      1710000100,
		DurationUs:   800,
		Status:       SpanStatusOK,
	}
	childData, err := json.Marshal(child)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(childData, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ParentSpanID == nil || *decoded.ParentSpanID != parentID {
		t.Fatalf("parent_span_id = %v, want %q", decoded.ParentSpanID, parentID)
	}
}
