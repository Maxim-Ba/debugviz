package scanner

import (
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func TestScanDemoWorker(t *testing.T) {
	root := repoRoot(t)

	graph, err := Scan([]string{"./demo/worker"}, Options{
		Dir:       root,
		Framework: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}

	var workerEntries int
	for _, node := range graph.Nodes {
		if node.Type != protocol.NodeTypeEntryPoint || node.Kind != protocol.EntryKindWorker {
			continue
		}
		workerEntries++

		job, _ := node.Metadata["job"].(string)
		if job != "OrderConsumer.Process" {
			t.Fatalf("unexpected worker job %q", job)
		}
		if node.Name != "OrderConsumer.Process" {
			t.Fatalf("worker entry name = %q", node.Name)
		}
		queue, _ := node.Metadata["queue"].(string)
		if queue != "orders" {
			t.Fatalf("worker queue = %q, want orders", queue)
		}
	}
	if workerEntries < 1 {
		t.Fatal("expected at least 1 worker entry in demo/worker graph")
	}

	entryID := entryNodeID(protocol.EntryKindWorker, "OrderConsumer.Process", "")
	wantHandler := "func:demo/worker/internal/consumer/order.go:Process"
	handlerLinked := false
	for _, edge := range graph.Edges {
		if edge.Type != protocol.EdgeTypeEntryHandles || edge.Source != entryID {
			continue
		}
		if edge.Target == wantHandler {
			handlerLinked = true
			break
		}
	}
	if !handlerLinked {
		t.Fatalf("missing entry_handles edge %s -> %s", entryID, wantHandler)
	}
}
