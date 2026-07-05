import type { EntryKind, Graph, Node } from "@debugviz/protocol";

export const ENTRY_KINDS: EntryKind[] = ["http", "grpc", "cli", "worker"];

export const ENTRY_KIND_LABELS: Record<EntryKind, string> = {
  http: "HTTP",
  grpc: "gRPC",
  cli: "CLI",
  worker: "Worker",
};

export function listEntryPoints(graph: Graph, kind?: EntryKind): Node[] {
  const entries = graph.nodes.filter((node) => node.type === "entry_point");
  const filtered = kind ? entries.filter((node) => node.kind === kind) : entries;
  return filtered.sort((a, b) => {
    const kindOrder = ENTRY_KINDS.indexOf(a.kind ?? "http") - ENTRY_KINDS.indexOf(b.kind ?? "http");
    if (kindOrder !== 0) {
      return kindOrder;
    }
    return a.name.localeCompare(b.name);
  });
}

export function entryKindCounts(graph: Graph): Record<EntryKind, number> {
  const counts: Record<EntryKind, number> = {
    http: 0,
    grpc: 0,
    cli: 0,
    worker: 0,
  };
  for (const node of graph.nodes) {
    if (node.type !== "entry_point" || !node.kind) {
      continue;
    }
    counts[node.kind] += 1;
  }
  return counts;
}

export function entryPointSubtitle(node: Node): string {
  const meta = node.metadata ?? {};
  if (node.kind === "http") {
    const method = String(meta.method ?? "");
    const path = String(meta.path ?? "");
    if (method && path) {
      return `${method} ${path}`;
    }
  }
  if (node.kind === "grpc") {
    const service = String(meta.service ?? "");
    const method = String(meta.method ?? "");
    if (service && method) {
      return `${service}/${method}`;
    }
  }
  if (node.kind === "cli") {
    const command = String(meta.command ?? "");
    if (command) {
      return command;
    }
  }
  if (node.kind === "worker") {
    const job = String(meta.job ?? "");
    const queue = String(meta.queue ?? "");
    if (job && queue) {
      return `${job} · ${queue}`;
    }
    if (job) {
      return job;
    }
  }
  return node.id;
}
