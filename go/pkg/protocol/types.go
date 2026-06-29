package protocol

// Graph schema version constant.
const GraphVersion = "2"

// NodeType identifies a node kind in the static graph.
type NodeType string

const (
	NodeTypePackage    NodeType = "package"
	NodeTypeFile       NodeType = "file"
	NodeTypeFunction   NodeType = "function"
	NodeTypeEntryPoint NodeType = "entry_point"
	NodeTypeMiddleware NodeType = "middleware"
)

// EntryKind identifies an entry point runtime category.
type EntryKind string

const (
	EntryKindHTTP   EntryKind = "http"
	EntryKindGRPC   EntryKind = "grpc"
	EntryKindCLI    EntryKind = "cli"
	EntryKindWorker EntryKind = "worker"
)

// EdgeType identifies an edge kind in the static graph.
type EdgeType string

const (
	EdgeTypeImports         EdgeType = "imports"
	EdgeTypeCalls           EdgeType = "calls"
	EdgeTypeEntryHandles    EdgeType = "entry_handles"
	EdgeTypeMiddlewareChain EdgeType = "middleware_chain"
)

// CallConfidence describes how a call edge was resolved.
type CallConfidence string

const (
	CallConfidenceStatic    CallConfidence = "static"
	CallConfidenceInterface CallConfidence = "interface"
	CallConfidenceUnknown   CallConfidence = "unknown"
)

// SpanStatus is the completion status of a trace span.
type SpanStatus string

const (
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
)

// Graph is the top-level static graph document (schemas/graph.schema.json).
type Graph struct {
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at,omitempty"`
	RootModule  string `json:"root_module,omitempty"`
	Nodes       []Node `json:"nodes"`
	Edges       []Edge `json:"edges"`
}

// Node is a vertex in the static graph.
type Node struct {
	ID       string         `json:"id"`
	Type     NodeType       `json:"type"`
	Name     string         `json:"name"`
	Kind     EntryKind      `json:"kind,omitempty"`
	Path     string         `json:"path,omitempty"`
	File     string         `json:"file,omitempty"`
	Line     int            `json:"line,omitempty"`
	Package  string         `json:"package,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Edge connects two nodes in the static graph.
type Edge struct {
	ID         string         `json:"id,omitempty"`
	Type       EdgeType       `json:"type"`
	Source     string         `json:"source"`
	Target     string         `json:"target"`
	Confidence CallConfidence `json:"confidence,omitempty"`
	Order      int            `json:"order,omitempty"`
}

// TraceEvent is a single span payload (schemas/trace-event.schema.json).
type TraceEvent struct {
	TraceID      string     `json:"trace_id"`
	SpanID       string     `json:"span_id"`
	ParentSpanID *string    `json:"parent_span_id"`
	Name         string     `json:"name"`
	File         string     `json:"file"`
	Line         int        `json:"line"`
	StartUs      int64      `json:"start_us"`
	DurationUs   int64      `json:"duration_us"`
	Status       SpanStatus `json:"status"`
	EntryKind    EntryKind  `json:"entry_kind,omitempty"`
	EntryName    string     `json:"entry_name,omitempty"`
}

// GraphMeta is returned by GET /api/graph/meta.
type GraphMeta struct {
	Nodes       int `json:"nodes"`
	Edges       int `json:"edges"`
	EntryPoints int `json:"entry_points"`
	Packages    int `json:"packages,omitempty"`
	Functions   int `json:"functions,omitempty"`
}

// ComputeGraphMeta derives stats from a graph document.
func ComputeGraphMeta(g *Graph) GraphMeta {
	meta := GraphMeta{
		Nodes: len(g.Nodes),
		Edges: len(g.Edges),
	}
	for _, node := range g.Nodes {
		switch node.Type {
		case NodeTypeEntryPoint:
			meta.EntryPoints++
		case NodeTypePackage:
			meta.Packages++
		case NodeTypeFunction:
			meta.Functions++
		}
	}
	return meta
}
