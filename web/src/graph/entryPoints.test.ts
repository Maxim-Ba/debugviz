import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import type { Graph } from "@debugviz/protocol";
import { entryKindCounts, listEntryPoints } from "./entryPoints.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const suiteGraph = JSON.parse(
  readFileSync(join(repoRoot, "schemas/examples/demo-suite-graph.json"), "utf8"),
) as Graph;

describe("entryPoints", () => {
  it("lists entry_point nodes sorted by kind and name", () => {
    const entries = listEntryPoints(suiteGraph);
    expect(entries.length).toBeGreaterThan(10);
    expect(entries.every((node) => node.type === "entry_point")).toBe(true);
    expect(entries[0]?.kind).toBe("http");
  });

  it("filters by kind", () => {
    const grpc = listEntryPoints(suiteGraph, "grpc");
    expect(grpc.length).toBe(3);
    expect(grpc.every((node) => node.kind === "grpc")).toBe(true);
  });

  it("counts entry kinds", () => {
    const counts = entryKindCounts(suiteGraph);
    expect(counts.http).toBeGreaterThan(0);
    expect(counts.grpc).toBe(3);
    expect(counts.cli).toBeGreaterThanOrEqual(2);
    expect(counts.worker).toBeGreaterThanOrEqual(1);
  });
});
