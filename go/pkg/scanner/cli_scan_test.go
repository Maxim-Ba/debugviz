package scanner

import (
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func TestScanDemoCLI(t *testing.T) {
	root := repoRoot(t)

	graph, err := Scan([]string{"./demo/cli"}, Options{
		Dir:       root,
		Framework: "cli",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantCommands := []string{"serve", "migrate up"}
	found := make(map[string]protocol.Node)
	for _, node := range graph.Nodes {
		if node.Type != protocol.NodeTypeEntryPoint || node.Kind != protocol.EntryKindCLI {
			continue
		}
		found[node.Name] = node
	}

	for _, want := range wantCommands {
		node, ok := found[want]
		if !ok {
			t.Fatalf("missing CLI entry %q", want)
		}
		command, _ := node.Metadata["command"].(string)
		if command != want {
			t.Fatalf("entry %q metadata command = %q", want, command)
		}
	}

	wantHandler := "func:demo/cli/main.go:runMigrateUp"
	entryID := entryNodeID(protocol.EntryKindCLI, "migrate up", "")
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
