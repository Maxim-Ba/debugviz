/**
 * Epic 2 smoke: graph upload, trace ingest, span mapping.
 * Run from repo root: pnpm --filter @debugviz/server smoke:epic2 graph.json
 */
import { existsSync, readFileSync } from "node:fs";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { buildServer } from "../src/app.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");

function resolveGraphPath(arg) {
  const fallback = join(repoRoot, "schemas", "examples", "demo-http-graph.json");
  if (!arg) {
    return fallback;
  }

  const candidates = isAbsolute(arg)
    ? [arg]
    : [resolve(process.cwd(), arg), resolve(repoRoot, arg)];

  for (const candidate of candidates) {
    if (existsSync(candidate)) {
      return candidate;
    }
  }

  throw new Error(
    `graph file not found: ${arg} (tried ${candidates.join(", ")})`,
  );
}

const graphPath = resolveGraphPath(process.argv[2]);

const sampleSpan = {
  trace_id: "smoke-trace",
  span_id: "smoke-root",
  parent_span_id: null,
  name: "handler.GetByID",
  file: "demo/http/internal/handler/users.go",
  line: 24,
  start_us: 1_710_000_000,
  duration_us: 1200,
  status: "ok",
  entry_kind: "http",
  entry_name: "GET /api/users/1",
};

const app = await buildServer();
console.log(`Using graph: ${graphPath}`);
const graph = readFileSync(graphPath, "utf8");

const checks = [];

function record(name, ok, detail = "") {
  checks.push({ name, ok, detail });
  const mark = ok ? "PASS" : "FAIL";
  console.log(`${mark} ${name}${detail ? `: ${detail}` : ""}`);
}

const upload = await app.inject({
  method: "POST",
  url: "/api/graph",
  payload: graph,
  headers: { "content-type": "application/json" },
});
record("2.1 POST /api/graph", upload.statusCode === 201, `status=${upload.statusCode}`);

const meta = await app.inject({ method: "GET", url: "/api/graph/meta" });
const metaBody = meta.json();
record(
  "2.1 GET /api/graph/meta",
  meta.statusCode === 200 && metaBody.entry_points > 0,
  JSON.stringify(metaBody),
);

const getGraph = await app.inject({ method: "GET", url: "/api/graph" });
record("2.1 GET /api/graph", getGraph.statusCode === 200 && getGraph.json().version === "2");

const ingest = await app.inject({
  method: "POST",
  url: "/api/traces/spans",
  payload: { spans: [sampleSpan] },
  headers: { "content-type": "application/json" },
});
record("2.2 POST /api/traces/spans", ingest.statusCode === 202, `accepted=${ingest.json().accepted}`);

const list = await app.inject({ method: "GET", url: "/api/traces" });
record("2.2 GET /api/traces", list.statusCode === 200 && list.json()[0]?.trace_id === "smoke-trace");

const detail = await app.inject({ method: "GET", url: "/api/traces/smoke-trace" });
const nodeId = detail.json().spans?.[0]?.node_id;
record(
  "2.3 span node_id mapping",
  nodeId === "entry:http:GET:/api/users/{id}",
  `node_id=${nodeId}`,
);

await app.listen({ port: 0, host: "127.0.0.1" });
const port = app.server.address().port;
const ws = new WebSocket(`ws://127.0.0.1:${port}/ws`);
let wsOk = false;
await new Promise((resolve, reject) => {
  const timer = setTimeout(() => reject(new Error("ws timeout")), 5000);
  ws.onmessage = (event) => {
    const msg = JSON.parse(String(event.data));
    if (msg.type === "ready") {
      void app
        .inject({
          method: "POST",
          url: "/api/traces/spans",
          payload: {
            spans: [{ ...sampleSpan, span_id: "ws-span", trace_id: "ws-trace" }],
          },
          headers: { "content-type": "application/json" },
        })
        .then((r) => {
          if (r.statusCode !== 202) reject(new Error(`ingest ${r.statusCode}`));
        });
      return;
    }
    if (msg.type === "span" && msg.trace_id === "ws-trace") {
      wsOk = true;
      clearTimeout(timer);
      resolve();
    }
  };
  ws.onerror = () => reject(new Error("ws error"));
});
ws.close();
record("2.2 WS /ws broadcast", wsOk);

await app.close();

const failed = checks.filter((c) => !c.ok);
if (failed.length) {
  console.error(`\n${failed.length} check(s) failed`);
  process.exit(1);
}
console.log(`\nAll ${checks.length} Epic 2 checks passed`);
