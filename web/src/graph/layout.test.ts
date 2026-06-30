import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import type { Graph } from "@debugviz/protocol";
import { computeForceLayout, syntheticGraph } from "./layout.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "..");
const demoGraph = JSON.parse(
  readFileSync(join(repoRoot, "schemas/examples/demo-http-graph.json"), "utf8"),
) as Graph;

describe("computeForceLayout (issue 3.2A)", () => {
  it("assigns a position to every node", () => {
    const { positions } = computeForceLayout(demoGraph, 40, 1);
    expect(positions.size).toBe(demoGraph.nodes.length);
    for (const node of demoGraph.nodes) {
      expect(positions.has(node.id)).toBe(true);
    }
  });

  it("lays out 500 nodes within 2s (issue 3.1 perf)", () => {
    const graph = syntheticGraph(500);
    const started = performance.now();
    const { positions } = computeForceLayout(graph, 60, 7);
    const elapsed = performance.now() - started;
    expect(positions.size).toBe(500);
    expect(elapsed).toBeLessThan(2000);
  });
});
