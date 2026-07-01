/**
 * Epic 6 smoke: cv-backend graph upload + HTTP path span mapping.
 * Run from repo root: pnpm --filter @debugviz/server smoke:epic6
 */
import { existsSync, readFileSync } from "node:fs";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { buildServer } from "../src/app.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");

function resolveGraphPath(arg) {
  const fallback = join(repoRoot, "examples", "cv-backend", "graph.json");
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
const graph = JSON.parse(readFileSync(graphPath, "utf8"));

const traceSpans = [
  {
    trace_id: "cv-smoke-trace",
    span_id: "cv-root",
    parent_span_id: null,
    name: "cv-backend GET /api/tech/1",
    file: "internal/router/techhndlr.go",
    line: 41,
    start_us: 1_710_000_000,
    duration_us: 2500,
    status: "ok",
    entry_kind: "http",
    entry_name: "GET /api/tech/1",
  },
  {
    trace_id: "cv-smoke-trace",
    span_id: "cv-handler",
    parent_span_id: "cv-root",
    name: "router.TechHandler.TechGet",
    file: "internal/router/techhndlr.go",
    line: 49,
    start_us: 1_710_000_100,
    duration_us: 2000,
    status: "ok",
  },
  {
    trace_id: "cv-smoke-trace",
    span_id: "cv-service",
    parent_span_id: "cv-handler",
    name: "services.TechService.GetWithTags",
    file: "internal/services/technology.go",
    line: 97,
    start_us: 1_710_000_300,
    duration_us: 1500,
    status: "ok",
  },
  {
    trace_id: "cv-smoke-trace",
    span_id: "cv-repo",
    parent_span_id: "cv-service",
    name: "repository.TechnologyRepo.GetWithTags",
    file: "internal/repository/technology.go",
    line: 230,
    start_us: 1_710_000_500,
    duration_us: 1000,
    status: "ok",
  },
];

const expectedNodeIds = {
  "cv-root": "entry:http:GET:/api/tech/{techID}",
  "cv-handler": "func:internal/router/techhndlr.go:TechGet",
  "cv-service": "func:internal/services/technology.go:GetWithTags",
  "cv-repo": "func:internal/repository/technology.go:GetWithTags",
};

const pathEdges = [
  ["entry:http:GET:/api/tech/{techID}", "func:internal/router/techhndlr.go:TechGet"],
  ["func:internal/router/techhndlr.go:TechGet", "func:internal/services/technology.go:GetWithTags"],
];

const app = await buildServer();
console.log(`Using graph: ${graphPath}`);

const checks = [];

function record(name, ok, detail = "") {
  checks.push({ name, ok, detail });
  const mark = ok ? "PASS" : "FAIL";
  console.log(`${mark} ${name}${detail ? `: ${detail}` : ""}`);
}

const entryPoints = graph.nodes.filter((node) => node.type === "entry_point");
record("6.2 cv-backend entry_points >= 20", entryPoints.length >= 20, `count=${entryPoints.length}`);

const upload = await app.inject({
  method: "POST",
  url: "/api/graph",
  payload: graph,
  headers: { "content-type": "application/json" },
});
record("6.3 POST /api/graph", upload.statusCode === 201, `status=${upload.statusCode}`);

const meta = await app.inject({ method: "GET", url: "/api/graph/meta" });
const metaBody = meta.json();
record(
  "6.3 GET /api/graph/meta",
  meta.statusCode === 200 && metaBody.entry_points >= 20,
  JSON.stringify(metaBody),
);

const ingest = await app.inject({
  method: "POST",
  url: "/api/traces/spans",
  payload: { spans: traceSpans },
  headers: { "content-type": "application/json" },
});
record("6.3 POST /api/traces/spans", ingest.statusCode === 202, `accepted=${ingest.json().accepted}`);

const detail = await app.inject({ method: "GET", url: "/api/traces/cv-smoke-trace" });
const storedSpans = detail.json().spans ?? [];

for (const span of traceSpans) {
  const stored = storedSpans.find((item) => item.span_id === span.span_id);
  const want = expectedNodeIds[span.span_id];
  record(
    `6.3 node_id ${span.span_id}`,
    stored?.node_id === want,
    `node_id=${stored?.node_id ?? "null"} want=${want}`,
  );
}

const edgeKeys = new Set(
  graph.edges.map((edge) => `${edge.source}→${edge.target}`),
);
for (const [source, target] of pathEdges) {
  const hasEntryHandles = edgeKeys.has(`${source}→${target}`);
  const hasCalls = graph.edges.some(
    (edge) =>
      edge.type === "calls" && edge.source === source && edge.target === target,
  );
  record(
    `6.3 path edge ${source.split(":").pop()} → ${target.split(":").pop()}`,
    hasEntryHandles || hasCalls,
    hasEntryHandles ? "entry_handles" : hasCalls ? "calls" : "missing",
  );
}

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
            spans: [{ ...traceSpans[0], span_id: "cv-ws", trace_id: "cv-ws-trace" }],
          },
          headers: { "content-type": "application/json" },
        })
        .then((r) => {
          if (r.statusCode !== 202) reject(new Error(`ingest ${r.statusCode}`));
        });
      return;
    }
    if (msg.type === "span" && msg.trace_id === "cv-ws-trace") {
      wsOk = msg.span?.node_id === expectedNodeIds["cv-root"];
      clearTimeout(timer);
      resolve();
    }
  };
  ws.onerror = () => reject(new Error("ws error"));
});
ws.close();
record("6.3 WS /ws cv-backend path", wsOk);

await app.close();

const failed = checks.filter((c) => !c.ok);
if (failed.length) {
  console.error(`\n${failed.length} check(s) failed`);
  process.exit(1);
}
console.log(`\nAll ${checks.length} Epic 6 checks passed`);
