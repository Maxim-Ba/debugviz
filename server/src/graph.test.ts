import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { buildServer } from "./app.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const demoGraphPath = join(repoRoot, "schemas", "examples", "demo-http-graph.json");
const demoGraph = readFileSync(demoGraphPath, "utf8");

describe("graph API (issue 2.1)", () => {
  it("POST /api/graph validates and stores graph v2", async () => {
    const app = await buildServer();

    const upload = await app.inject({
      method: "POST",
      url: "/api/graph",
      payload: demoGraph,
      headers: { "content-type": "application/json" },
    });
    expect(upload.statusCode).toBe(201);
    expect(upload.json()).toEqual({ ok: true });

    const graph = await app.inject({ method: "GET", url: "/api/graph" });
    expect(graph.statusCode).toBe(200);
    const body = graph.json();
    expect(body.version).toBe("2");
    expect(body.nodes.some((node: { type: string }) => node.type === "entry_point")).toBe(true);

    const meta = await app.inject({ method: "GET", url: "/api/graph/meta" });
    expect(meta.statusCode).toBe(200);
    expect(meta.json()).toMatchObject({
      nodes: body.nodes.length,
      edges: body.edges.length,
      entry_points: 2,
    });

    await app.close();
  });

  it("rejects invalid graph payload", async () => {
    const app = await buildServer();
    const response = await app.inject({
      method: "POST",
      url: "/api/graph",
      payload: { version: "1", nodes: [], edges: [] },
      headers: { "content-type": "application/json" },
    });
    expect(response.statusCode).toBe(400);
    expect(response.json().error).toBeTruthy();
    await app.close();
  });

  it("GET /api/graph returns 404 before upload", async () => {
    const app = await buildServer();
    const response = await app.inject({ method: "GET", url: "/api/graph" });
    expect(response.statusCode).toBe(404);
    await app.close();
  });
});
