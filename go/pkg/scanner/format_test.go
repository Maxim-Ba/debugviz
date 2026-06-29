package scanner

import (
	"strings"
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func TestFormatDOT(t *testing.T) {
	graph := &protocol.Graph{
		Version:    protocol.GraphVersion,
		RootModule: "example.com/app",
		Nodes: []protocol.Node{
			{ID: "pkg:example.com/app", Type: protocol.NodeTypePackage, Name: "app", Path: "app"},
			{ID: "entry:http:GET:/health", Type: protocol.NodeTypeEntryPoint, Name: "GET /health", Kind: protocol.EntryKindHTTP},
		},
		Edges: []protocol.Edge{
			{Type: protocol.EdgeTypeImports, Source: "pkg:example.com/app", Target: "pkg:fmt"},
		},
	}

	dot := FormatDOT(graph)
	for _, want := range []string{
		"digraph debugviz {",
		`"pkg:example.com/app"`,
		`"entry:http:GET:/health"`,
		`label="imports"`,
	} {
		if !strings.Contains(dot, want) {
			t.Fatalf("FormatDOT() missing %q\n%s", want, dot)
		}
	}
}
