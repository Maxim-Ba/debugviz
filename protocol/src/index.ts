/** Graph schema version (schemas/graph.schema.json). */
export const GRAPH_VERSION = "2" as const;

export type NodeType = "package" | "file" | "function" | "entry_point" | "middleware";

export type EntryKind = "http" | "grpc" | "cli" | "worker";

export type EdgeType = "imports" | "calls" | "entry_handles" | "middleware_chain";

export type CallConfidence = "static" | "interface" | "unknown";

export type SpanStatus = "ok" | "error";

/** Static graph document (schemas/graph.schema.json). */
export interface Graph {
  version: typeof GRAPH_VERSION | string;
  generated_at?: string;
  root_module?: string;
  nodes: Node[];
  edges: Edge[];
}

/** Vertex in the static graph. */
export interface Node {
  id: string;
  type: NodeType;
  name: string;
  kind?: EntryKind;
  path?: string;
  file?: string;
  line?: number;
  package?: string;
  metadata?: Record<string, unknown>;
}

/** Edge in the static graph. */
export interface Edge {
  id?: string;
  type: EdgeType;
  source: string;
  target: string;
  confidence?: CallConfidence;
  order?: number;
}

/** Single span payload (schemas/trace-event.schema.json). */
export interface TraceEvent {
  trace_id: string;
  span_id: string;
  parent_span_id?: string | null;
  name: string;
  file: string;
  line: number;
  start_us: number;
  duration_us: number;
  status: SpanStatus;
  entry_kind?: EntryKind;
  entry_name?: string;
}

/** Response shape for GET /api/graph/meta. */
export interface GraphMeta {
  nodes: number;
  edges: number;
  entry_points: number;
  packages?: number;
  functions?: number;
}

export function computeGraphMeta(graph: Graph): GraphMeta {
  const meta: GraphMeta = {
    nodes: graph.nodes.length,
    edges: graph.edges.length,
    entry_points: 0,
    packages: 0,
    functions: 0,
  };

  for (const node of graph.nodes) {
    switch (node.type) {
      case "entry_point":
        meta.entry_points += 1;
        break;
      case "package":
        meta.packages = (meta.packages ?? 0) + 1;
        break;
      case "function":
        meta.functions = (meta.functions ?? 0) + 1;
        break;
    }
  }

  return meta;
}
