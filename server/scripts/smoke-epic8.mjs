/**
 * Epic 8 smoke: multi-runtime demos — gRPC grpcurl + worker auto-trace in Docker stack.
 * Run: make docker-smoke-epic8  OR  node server/scripts/smoke-epic8.mjs --docker
 */
import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const dockerMode = process.argv.includes("--docker");

const SERVER_URL = process.env.SERVER_URL ?? "http://localhost:4000";
const GRPC_ADDR = process.env.GRPC_ADDR ?? "localhost:9090";

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

async function waitForTrace(predicate, attempts = 40) {
  for (let i = 0; i < attempts; i++) {
    const list = await (await fetch(`${SERVER_URL}/api/traces`)).json();
    if (!Array.isArray(list)) {
      await new Promise((r) => setTimeout(r, 500));
      continue;
    }
    for (const item of list) {
      const detail = await (await fetch(`${SERVER_URL}/api/traces/${item.trace_id}`)).json();
      const spans = Array.isArray(detail.spans) ? detail.spans : [];
      const hit = spans.find(predicate);
      if (hit) {
        return { trace: detail, span: hit };
      }
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  return null;
}

function runGrpcurl(args) {
  const local = spawnSync("grpcurl", args, {
    cwd: repoRoot,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (!local.error || local.error.code !== "ENOENT") {
    return {
      ok: local.status === 0,
      stdout: local.stdout ?? "",
      stderr: local.stderr ?? "",
      status: local.status,
      error: local.error,
    };
  }

  const network = process.env.DEBUGVIZ_DOCKER_NETWORK ?? "debugviz_default";
  const dockerArgs = ["run", "--rm", `--network=${network}`, "fullstorydev/grpcurl"];
  const targetIdx = args.indexOf(GRPC_ADDR);
  const remoteArgs =
    targetIdx >= 0
      ? [...args.slice(0, targetIdx), "demo-grpc:9090", ...args.slice(targetIdx + 1)]
      : args;

  const remote = spawnSync("docker", [...dockerArgs, ...remoteArgs], {
    cwd: repoRoot,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  return {
    ok: remote.status === 0,
    stdout: remote.stdout ?? "",
    stderr: remote.stderr ?? "",
    status: remote.status,
    error: remote.error,
  };
}

async function smokeDockerMultiRuntime() {
  console.log(`Epic 8 docker: server=${SERVER_URL} grpc=${GRPC_ADDR}`);

  const health = await waitFor(SERVER_URL, "/health");
  record("8.x server /health", health.ok, `status=${health.status}`);

  const metaRes = await waitFor(SERVER_URL, "/api/graph/meta");
  const meta = await metaRes.json();
  const kinds = new Set();
  const graph = await (await fetch(`${SERVER_URL}/api/graph`)).json();
  for (const node of graph.nodes ?? []) {
    if (node.type === "entry_point" && node.kind) {
      kinds.add(node.kind);
    }
  }
  record(
    "8.x suite graph loaded",
    meta.entry_points >= 10 && kinds.has("http") && kinds.has("grpc") && kinds.has("worker"),
    `entry_points=${meta.entry_points} kinds=${[...kinds].join(",")}`,
  );

  const grpcList = runGrpcurl(["-plaintext", GRPC_ADDR, "list"]);
  record(
    "8.1 grpcurl list",
    grpcList.ok,
    grpcList.ok ? grpcList.stdout.trim().split("\n")[0] : grpcList.stderr || String(grpcList.error),
  );

  const beforeTraces = await (await fetch(`${SERVER_URL}/api/traces`)).json();
  const beforeCount = Array.isArray(beforeTraces) ? beforeTraces.length : 0;

  const grpcCall = runGrpcurl([
    "-plaintext",
    "-H",
    "x-trace-id: epic8-grpc-demo",
    "-d",
    "{}",
    GRPC_ADDR,
    "user.v1.UserService/GetUser",
  ]);
  record("8.1 grpcurl GetUser", grpcCall.ok, grpcCall.ok ? "ok" : grpcCall.stderr || String(grpcCall.error));

  const grpcHit = await waitForTrace(
    (s) => s.entry_kind === "grpc" && s.entry_name === "user.v1.UserService/GetUser",
    20,
  );
  record(
    "8.1 gRPC trace ingested",
    grpcHit !== null,
    grpcHit ? `trace_id=${grpcHit.trace.trace_id}` : `traces before=${beforeCount}`,
  );
  if (grpcHit) {
    record(
      "8.1 gRPC span mapped to entry node",
      grpcHit.span.node_id === "entry:grpc:user.v1.UserService:GetUser",
      `node_id=${grpcHit.span.node_id}`,
    );
  }

  const workerHit = await waitForTrace(
    (s) => s.entry_kind === "worker" && s.entry_name === "OrderConsumer.Process",
    30,
  );
  record(
    "8.3 worker trace ingested",
    workerHit !== null,
    workerHit ? `trace_id=${workerHit.trace.trace_id}` : "no worker span yet",
  );
  if (workerHit) {
    record(
      "8.3 worker span mapped to entry node",
      workerHit.span.node_id === "entry:worker:OrderConsumer.Process",
      `node_id=${workerHit.span.node_id}`,
    );
  }
}

if (!dockerMode) {
  console.error("smoke-epic8 requires --docker (multi-runtime stack)");
  process.exit(1);
}

await smokeDockerMultiRuntime();

const failed = checks.filter((c) => !c.ok);
if (failed.length) {
  console.error(`\n${failed.length} check(s) failed`);
  process.exit(1);
}
console.log(`\nAll ${checks.length} Epic 8 checks passed`);
