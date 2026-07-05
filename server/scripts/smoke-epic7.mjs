/**
 * Epic 7 smoke: Docker one-liner — init graph on server start + live demo/http trace.
 * In-process: pnpm --filter @debugviz/server smoke:epic7  (needs tsx + server deps)
 * Live stack: make docker-smoke  OR  node server/scripts/smoke-epic7.mjs --docker
 */
import { existsSync } from "node:fs";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const dockerMode = process.argv.includes("--docker");

const SERVER_URL = process.env.SERVER_URL ?? "http://localhost:4000";
const DEMO_URL = process.env.DEMO_URL ?? "http://localhost:8080";
const UI_URL = process.env.UI_URL ?? "http://localhost:3000";

function resolveGraphPath() {
  const fallback = join(repoRoot, "schemas", "examples", "demo-http-graph.json");
  const arg = process.argv.find((a) => a.endsWith(".json"));
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

  throw new Error(`graph file not found: ${arg}`);
}

const checks = [];

function record(name, ok, detail = "") {
  checks.push({ name, ok, detail });
  const mark = ok ? "PASS" : "FAIL";
  console.log(`${mark} ${name}${detail ? `: ${detail}` : ""}`);
}

async function waitFor(url, path, attempts = 60) {
  const target = `${url}${path}`;
  for (let i = 0; i < attempts; i++) {
    try {
      const res = await fetch(target);
      if (res.ok) {
        return res;
      }
    } catch {
      // retry
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error(`timeout waiting for ${target}`);
}

async function smokeInitGraph() {
  const { buildServer } = await import("../src/app.js");
  const graphPath = resolveGraphPath();
  process.env.INIT_GRAPH_PATH = graphPath;

  const app = await buildServer();
  console.log(`Using INIT_GRAPH_PATH=${graphPath}`);

  const meta = await app.inject({ method: "GET", url: "/api/graph/meta" });
  const metaBody = meta.json();
  record(
    "7.1 init graph meta",
    meta.statusCode === 200 && metaBody.entry_points > 0,
    JSON.stringify(metaBody),
  );

  const getGraph = await app.inject({ method: "GET", url: "/api/graph" });
  record(
    "7.1 init graph body",
    getGraph.statusCode === 200 && getGraph.json().version === "2",
    `status=${getGraph.statusCode}`,
  );

  await app.close();
}

async function smokeDockerStack() {
  console.log(`Docker stack: server=${SERVER_URL} demo=${DEMO_URL} ui=${UI_URL}`);

  const health = await waitFor(SERVER_URL, "/health");
  record("7.1 server /health", health.ok, `status=${health.status}`);

  const metaRes = await waitFor(SERVER_URL, "/api/graph/meta");
  const meta = await metaRes.json();
  record(
    "7.1 docker graph preloaded",
    meta.entry_points > 0,
    JSON.stringify(meta),
  );

  const demoHealth = await waitFor(DEMO_URL, "/health");
  const demoBody = await demoHealth.text();
  record("7.1 demo-http /health", demoHealth.ok && demoBody.includes("ok"), demoBody.trim());

  const traceID = `epic7-smoke-${Date.now()}`;
  const ping = await fetch(`${DEMO_URL}/api/users/1`, {
    headers: { "X-Trace-ID": traceID },
  });
  record("7.1 demo-http GET /api/users/1", ping.ok, `status=${ping.status}`);

  let detail = null;
  for (let i = 0; i < 20; i++) {
    await new Promise((r) => setTimeout(r, 250));
    const res = await fetch(`${SERVER_URL}/api/traces/${traceID}`);
    if (res.ok) {
      detail = await res.json();
      break;
    }
  }
  record(
    "7.1 live trace ingested",
    detail !== null && Array.isArray(detail.spans) && detail.spans.length > 0,
    detail ? `spans=${detail.spans.length}` : "trace not found",
  );

  if (detail?.spans?.length) {
    const spans = detail.spans;
    const httpSpan = spans.find((s) => s.entry_kind === "http");
    const nodeId = httpSpan?.node_id ?? null;
    record(
      "7.1 span mapped to entry node",
      nodeId === "entry:http:GET:/api/users/{id}",
      `node_id=${nodeId}`,
    );

    const serviceSpan = spans.find((s) => s.name?.includes("UserService.GetByID"));
    record(
      "7.1 service span mapped",
      serviceSpan?.node_id === "func:demo/http/internal/service/user.go:GetByID",
      `node_id=${serviceSpan?.node_id ?? "null"}`,
    );

    const repoSpan = spans.find((s) => s.name?.includes("UserRepository.FindByID"));
    record(
      "7.1 repository span mapped",
      repoSpan?.node_id === "func:demo/http/internal/repository/user.go:FindByID",
      `node_id=${repoSpan?.node_id ?? "null"}`,
    );

    const mappedPath = [httpSpan, serviceSpan, repoSpan].filter(
      (span) => span?.node_id,
    ).length;
    record(
      "7.1 HTTP live path nodes",
      mappedPath >= 3,
      `mapped=${mappedPath}/3`,
    );
  }

  const ui = await waitFor(UI_URL, "/");
  record("7.1 UI reachable", ui.ok, `status=${ui.status}`);
}

if (dockerMode) {
  await smokeDockerStack();
} else {
  await smokeInitGraph();
}

const failed = checks.filter((c) => !c.ok);
if (failed.length) {
  console.error(`\n${failed.length} check(s) failed`);
  process.exit(1);
}
console.log(`\nAll ${checks.length} Epic 7 checks passed`);
