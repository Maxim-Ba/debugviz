import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, describe, expect, it } from "vitest";
import { buildServer } from "./app.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const graphPath = join(repoRoot, "schemas", "examples", "demo-http-graph.json");

describe("buildServer", () => {
  afterEach(() => {
    delete process.env.INIT_GRAPH_PATH;
  });

  it("responds on /health", async () => {
    const app = await buildServer();
    const response = await app.inject({ method: "GET", url: "/health" });
    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual({ status: "ok" });
    await app.close();
  });

  it("loads graph from INIT_GRAPH_PATH", async () => {
    if (!existsSync(graphPath)) {
      return;
    }

    process.env.INIT_GRAPH_PATH = graphPath;
    const app = await buildServer();
    const meta = await app.inject({ method: "GET", url: "/api/graph/meta" });
    expect(meta.statusCode).toBe(200);
    expect(meta.json().entry_points).toBeGreaterThan(0);
    await app.close();
  });
});
