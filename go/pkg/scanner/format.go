package scanner

import (
	"fmt"
	"strings"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// FormatDOT renders a static graph as a Graphviz DOT document.
func FormatDOT(g *protocol.Graph) string {
	if g == nil {
		return "digraph debugviz {\n}\n"
	}

	var b strings.Builder
	b.WriteString("digraph debugviz {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [fontname=\"Helvetica\"];\n")

	for _, node := range g.Nodes {
		label := dotLabel(node)
		shape := dotShape(node.Type)
		fmt.Fprintf(&b, "  %s [label=%s, shape=%s];\n",
			dotID(node.ID), dotString(label), shape)
	}

	for _, edge := range g.Edges {
		label := string(edge.Type)
		if edge.Confidence != "" {
			label += " (" + string(edge.Confidence) + ")"
		}
		fmt.Fprintf(&b, "  %s -> %s [label=%s];\n",
			dotID(edge.Source), dotID(edge.Target), dotString(label))
	}

	b.WriteString("}\n")
	return b.String()
}

func dotLabel(node protocol.Node) string {
	switch node.Type {
	case protocol.NodeTypeEntryPoint:
		return node.Name
	case protocol.NodeTypeFunction, protocol.NodeTypeMiddleware:
		if node.File != "" {
			return node.Name + "\n" + node.File
		}
		return node.Name
	case protocol.NodeTypeFile:
		return node.Path
	case protocol.NodeTypePackage:
		if node.Path != "" {
			return node.Path
		}
		return node.Name
	default:
		return node.Name
	}
}

func dotShape(nodeType protocol.NodeType) string {
	switch nodeType {
	case protocol.NodeTypePackage:
		return "box"
	case protocol.NodeTypeFile:
		return "note"
	case protocol.NodeTypeFunction:
		return "ellipse"
	case protocol.NodeTypeEntryPoint:
		return "diamond"
	case protocol.NodeTypeMiddleware:
		return "hexagon"
	default:
		return "ellipse"
	}
}

func dotID(id string) string {
	return dotString(id)
}

func dotString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return `"` + value + `"`
}
