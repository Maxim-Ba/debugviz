import type { EntryKind, Graph, Node, TraceEvent } from "./index.js";

export interface SpanMapper {
  map(span: TraceEvent): string | null;
}

interface FileFunction {
  id: string;
  name: string;
  line: number;
}

function normalizePath(file: string): string {
  return file.replace(/\\/g, "/").replace(/^\.\//, "");
}

function shortName(qualified: string): string {
  const parts = qualified.split(".");
  return parts[parts.length - 1] ?? qualified;
}

function parseHttpEntryName(entryName: string): { method: string; path: string } | null {
  const trimmed = entryName.trim();
  const space = trimmed.indexOf(" ");
  if (space <= 0) {
    return null;
  }
  return {
    method: trimmed.slice(0, space).toUpperCase(),
    path: trimmed.slice(space + 1),
  };
}

function splitPath(path: string): string[] {
  return path.split("/").filter((segment) => segment.length > 0);
}

function matchHttpPath(pattern: string, actual: string): boolean {
  const patternParts = splitPath(pattern);
  const actualParts = splitPath(actual);
  if (patternParts.length !== actualParts.length) {
    return false;
  }
  return patternParts.every((part, index) => {
    if (part.startsWith("{") && part.endsWith("}")) {
      return actualParts[index] !== undefined;
    }
    return part === actualParts[index];
  });
}

function metadataString(node: Node, key: string): string {
  const value = node.metadata?.[key];
  return typeof value === "string" ? value : "";
}

function matchEntryPoint(node: Node, kind: EntryKind, entryName: string): boolean {
  if (node.type !== "entry_point" || node.kind !== kind) {
    return false;
  }

  switch (kind) {
    case "http": {
      const parsed = parseHttpEntryName(entryName);
      if (!parsed) {
        return node.name === entryName;
      }
      const method = metadataString(node, "method").toUpperCase();
      const path = metadataString(node, "path");
      return method === parsed.method && matchHttpPath(path, parsed.path);
    }
    case "grpc": {
      const service = metadataString(node, "service");
      const method = metadataString(node, "method");
      if (service && method) {
        return entryName === `${service}/${method}` || entryName === `${service}.${method}`;
      }
      return node.name === entryName;
    }
    case "cli":
    case "worker": {
      const command = metadataString(node, "command") || metadataString(node, "job");
      return command === entryName || node.name === entryName || node.id.endsWith(`:${entryName}`);
    }
    default:
      return node.name === entryName;
  }
}

function isRootSpan(span: TraceEvent): boolean {
  return span.parent_span_id === null || span.parent_span_id === undefined;
}

export class GraphSpanMapper implements SpanMapper {
  private readonly entryPoints: Node[];
  private readonly functionsByFile = new Map<string, FileFunction[]>();
  private readonly functionsByName = new Map<string, string[]>();

  constructor(graph: Graph) {
    this.entryPoints = graph.nodes.filter((node) => node.type === "entry_point");

    for (const node of graph.nodes) {
      if (node.type !== "function" || !node.file) {
        continue;
      }
      const file = normalizePath(node.file);
      const bucket = this.functionsByFile.get(file) ?? [];
      bucket.push({ id: node.id, name: node.name, line: node.line ?? 0 });
      this.functionsByFile.set(file, bucket);

      const names = this.functionsByName.get(node.name) ?? [];
      names.push(node.id);
      this.functionsByName.set(node.name, names);
    }

    for (const bucket of this.functionsByFile.values()) {
      bucket.sort((a, b) => a.line - b.line);
    }
  }

  map(span: TraceEvent): string | null {
    if (isRootSpan(span) && span.entry_kind && span.entry_name) {
      const entry = this.entryPoints.find((node) =>
        matchEntryPoint(node, span.entry_kind!, span.entry_name!),
      );
      if (entry) {
        return entry.id;
      }
    }

    const byLocation = this.mapByFileLine(span.file, span.line);
    if (byLocation) {
      return byLocation;
    }

    return this.mapByName(span.name, span.file);
  }

  private mapByFileLine(file: string, line: number): string | null {
    const normalized = normalizePath(file);
    const candidates =
      this.functionsByFile.get(normalized) ??
      [...this.functionsByFile.entries()]
        .filter(([path]) => normalized.endsWith(path) || path.endsWith(normalized))
        .flatMap(([, items]) => items);

    if (!candidates.length) {
      return null;
    }

    let best: FileFunction | null = null;
    for (const candidate of candidates) {
      if (candidate.line <= line && (!best || candidate.line > best.line)) {
        best = candidate;
      }
    }
    return best?.id ?? null;
  }

  private mapByName(name: string, file: string): string | null {
    const exact = this.functionsByName.get(name);
    if (exact?.length === 1) {
      return exact[0] ?? null;
    }

    const short = shortName(name);
    let byShort = this.functionsByName.get(short);
    if (!byShort?.length) {
      return null;
    }

    const normalized = normalizePath(file);
    if (normalized && normalized !== "unknown.go") {
      const inFile = byShort.filter((id) => {
        const funcFile = id.slice("func:".length, id.lastIndexOf(":"));
        return normalized.endsWith(funcFile) || funcFile.endsWith(normalized);
      });
      if (inFile.length === 1) {
        return inFile[0] ?? null;
      }
      if (inFile.length > 1) {
        byShort = inFile;
      }
    }

    if (byShort.length === 1) {
      return byShort[0] ?? null;
    }

    const prefix = name.includes(".") ? name.slice(0, name.lastIndexOf(".")).toLowerCase() : "";
    if (prefix) {
      const segments = prefix.split(".").filter(Boolean);
      const hinted = byShort.find((id) => {
        const lower = id.toLowerCase();
        return segments.some((segment) => lower.includes(`/${segment}/`) || lower.includes(`${segment}.`));
      });
      if (hinted) {
        return hinted;
      }
    }

    return byShort[0] ?? null;
  }
}

export function mapSpanToNode(graph: Graph | null, span: TraceEvent): string | null {
  if (!graph) {
    return null;
  }
  return new GraphSpanMapper(graph).map(span);
}
