import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { buildServer } from "./app.js";
import type { TraceBroadcastMessage } from "./trace/store.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const demoGraph = readFileSync(
  join(repoRoot, "schemas", "examples", "demo-http-graph.json"),
  "utf8",
);

const sampleSpan = {
  trace_id: "trace-1",
  span_id: "span-root",
  parent_span_id: null,
  name: "handler.GetByID",
  file: "demo/http/internal/handler/users.go",
  line: 24,
  start_us: 1_710_000_000,
  duration_us: 1200,
  status: "ok" as const,
  entry_kind: "http" as const,
  entry_name: "GET /api/users/1",
};

describe("trace API (issue 2.2)", () => {
  it("ingests spans and lists traces", async () => {
    const app = await buildServer();

    const ingest = await app.inject({
      method: "POST",
      url: "/api/traces/spans",
      payload: { spans: [sampleSpan] },
      headers: { "content-type": "application/json" },
    });
    expect(ingest.statusCode).toBe(202);
    expect(ingest.json()).toEqual({ accepted: 1 });

    const list = await app.inject({ method: "GET", url: "/api/traces" });
    expect(list.statusCode).toBe(200);
    expect(list.json()).toEqual([
      {
        trace_id: "trace-1",
        span_count: 1,
        started_at_us: sampleSpan.start_us,
        status: "ok",
        entry_kind: "http",
        entry_name: "GET /api/users/1",
      },
    ]);

    const detail = await app.inject({ method: "GET", url: "/api/traces/trace-1" });
    expect(detail.statusCode).toBe(200);
    expect(detail.json().spans).toHaveLength(1);

    await app.close();
  });

  it("broadcasts ingested spans over /ws", async () => {
    const app = await buildServer();
    await app.listen({ port: 0, host: "127.0.0.1" });
    const address = app.server.address();
    if (!address || typeof address === "string") {
      throw new Error("expected bound TCP port");
    }

    const ws = new WebSocket(`ws://127.0.0.1:${address.port}/ws`);
    const messages: TraceBroadcastMessage[] = [];

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error("ws timeout")), 3000);
      ws.onmessage = (event) => {
        const payload = JSON.parse(String(event.data)) as { type: string };
        if (payload.type === "ready") {
          void app
            .inject({
              method: "POST",
              url: "/api/traces/spans",
              payload: { spans: [sampleSpan] },
              headers: { "content-type": "application/json" },
            })
            .then((response) => {
              expect(response.statusCode).toBe(202);
            });
          return;
        }
        messages.push(payload as TraceBroadcastMessage);
        clearTimeout(timeout);
        resolve();
      };
      ws.onerror = () => reject(new Error("ws error"));
    });

    expect(messages).toHaveLength(1);
    expect(messages[0]?.span.span_id).toBe("span-root");

    ws.close();
    await app.close();
  });

  it("maps spans to graph nodes when graph is loaded (issue 2.3)", async () => {
    const app = await buildServer();

    const upload = await app.inject({
      method: "POST",
      url: "/api/graph",
      payload: demoGraph,
      headers: { "content-type": "application/json" },
    });
    expect(upload.statusCode).toBe(201);

    const ingest = await app.inject({
      method: "POST",
      url: "/api/traces/spans",
      payload: { spans: [sampleSpan] },
      headers: { "content-type": "application/json" },
    });
    expect(ingest.statusCode).toBe(202);

    const detail = await app.inject({ method: "GET", url: "/api/traces/trace-1" });
    expect(detail.json().spans[0].node_id).toBe("entry:http:GET:/api/users/{id}");

    await app.close();
  });
});
