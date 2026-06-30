import type { Edge, EntryKind, Node, NodeType } from "@debugviz/protocol";

const HTTP_METHOD_COLORS: Record<string, number> = {
  GET: 0x22c55e,
  POST: 0x3b82f6,
  PUT: 0xf59e0b,
  PATCH: 0xa855f7,
  DELETE: 0xef4444,
  HEAD: 0x64748b,
  OPTIONS: 0x64748b,
};

const ENTRY_KIND_COLORS: Record<EntryKind, number> = {
  http: 0x38bdf8,
  grpc: 0xa78bfa,
  cli: 0xfb923c,
  worker: 0xfacc15,
};

const NODE_TYPE_COLORS: Record<NodeType, number> = {
  package: 0x6366f1,
  file: 0x0ea5e9,
  function: 0x94a3b8,
  entry_point: 0x38bdf8,
  middleware: 0xf472b6,
};

export function nodeColor(node: Node): number {
  if (node.type === "entry_point" && node.kind) {
    if (node.kind === "http") {
      const method = String(node.metadata?.method ?? "GET").toUpperCase();
      return HTTP_METHOD_COLORS[method] ?? ENTRY_KIND_COLORS.http;
    }
    return ENTRY_KIND_COLORS[node.kind];
  }
  return NODE_TYPE_COLORS[node.type];
}

export function nodeScale(node: Node): number {
  switch (node.type) {
    case "package":
      return 1.1;
    case "file":
      return 0.75;
    case "entry_point":
      return 1;
    case "middleware":
      return 0.55;
    default:
      return 0.45;
  }
}

export function nodeGeometryKind(node: Node): "sphere" | "box" | "octahedron" {
  switch (node.type) {
    case "package":
      return "sphere";
    case "file":
      return "box";
    case "entry_point":
      return "octahedron";
    default:
      return "box";
  }
}

export function edgeColor(edge: Edge): number {
  switch (edge.type) {
    case "calls":
      return 0xfbbf24;
    case "entry_handles":
      return 0x34d399;
    case "middleware_chain":
      return 0xf472b6;
    default:
      return 0x475569;
  }
}

export function edgeWidth(edge: Edge): number {
  switch (edge.type) {
    case "calls":
    case "entry_handles":
      return 1.5;
    default:
      return 0.6;
  }
}

export type LodLevel = "package" | "file" | "detail";

export function lodLevelForDistance(distance: number): LodLevel {
  if (distance > 55) {
    return "package";
  }
  if (distance > 28) {
    return "file";
  }
  return "detail";
}

export function isNodeVisibleAtLod(node: Node, lod: LodLevel): boolean {
  switch (lod) {
    case "package":
      return node.type === "package" || node.type === "entry_point";
    case "file":
      return node.type !== "function" && node.type !== "middleware";
    case "detail":
      return true;
  }
}

export const HIGHLIGHT_COLOR = 0xff6b35;
export const ERROR_COLOR = 0xef4444;
export const DIM_COLOR = 0x1e293b;
