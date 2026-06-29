import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { computeGraphMeta, type Graph, type TraceEvent } from "./index.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const demoGraphPath = join(repoRoot, "schemas", "examples", "demo-http-graph.json");

describe("protocol types", () => {
  it("loads demo-http-graph.json with entry_point nodes", () => {
    const graph = JSON.parse(readFileSync(demoGraphPath, "utf8")) as Graph;

    expect(graph.version).toBe("2");
    expect(graph.nodes.length).toBeGreaterThan(0);
    expect(graph.edges.length).toBeGreaterThan(0);

    const entries = graph.nodes.filter((node) => node.type === "entry_point");
    expect(entries).toHaveLength(2);
    expect(entries[0]?.kind).toBe("http");
    expect(entries[0]?.metadata).toBeDefined();

    const meta = computeGraphMeta(graph);
    expect(meta.entry_points).toBe(2);
    expect(meta.nodes).toBe(graph.nodes.length);
  });

  it("accepts trace event shape from README", () => {
    const event: TraceEvent = {
      trace_id: "trace-1",
      span_id: "span-root",
      parent_span_id: null,
      name: "UserService.GetByID",
      file: "internal/services/user.go",
      line: 42,
      start_us: 1710000000,
      duration_us: 1200,
      status: "ok",
      entry_kind: "http",
      entry_name: "GET /api/users/1",
    };

    expect(event.entry_kind).toBe("http");
  });
});
