import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import type { Graph, TraceEvent } from "./index.js";
import { GraphSpanMapper } from "./span-mapper.js";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const demoGraph = JSON.parse(
  readFileSync(join(repoRoot, "go/pkg/scanner/testdata/demo_http.golden.json"), "utf8"),
) as Graph;

describe("GraphSpanMapper (issue 2.3)", () => {
  const mapper = new GraphSpanMapper(demoGraph);

  it("maps HTTP root span to entry_point via path pattern", () => {
    const span: TraceEvent = {
      trace_id: "t1",
      span_id: "s1",
      parent_span_id: null,
      name: "HTTP GET /api/users/1",
      file: "demo/http/internal/handler/users.go",
      line: 24,
      start_us: 1,
      duration_us: 1,
      status: "ok",
      entry_kind: "http",
      entry_name: "GET /api/users/42",
    };

    expect(mapper.map(span)).toBe("entry:http:GET:/api/users/{id}");
  });

  it("maps inner span by file and line", () => {
    const span: TraceEvent = {
      trace_id: "t1",
      span_id: "s2",
      parent_span_id: "s1",
      name: "UserService.GetByID",
      file: "demo/http/internal/service/user.go",
      line: 19,
      start_us: 2,
      duration_us: 1,
      status: "ok",
    };

    expect(mapper.map(span)).toBe("func:demo/http/internal/service/user.go:GetByID");
  });

  it("falls back to function name", () => {
    const span: TraceEvent = {
      trace_id: "t1",
      span_id: "s3",
      parent_span_id: "s1",
      name: "UserHandler.GetByID",
      file: "demo/http/internal/handler/users.go",
      line: 1,
      start_us: 3,
      duration_us: 1,
      status: "ok",
    };

    expect(mapper.map(span)).toBe("func:demo/http/internal/handler/users.go:GetByID");
  });

  it("maps gRPC root span to entry_point", () => {
    const graph: Graph = {
      version: "2",
      nodes: [
        {
          id: "entry:grpc:UserService:GetByID",
          type: "entry_point",
          kind: "grpc",
          name: "UserService/GetByID",
          metadata: { service: "UserService", method: "GetByID" },
        },
      ],
      edges: [],
    };
    const grpcMapper = new GraphSpanMapper(graph);
    const span: TraceEvent = {
      trace_id: "t2",
      span_id: "s1",
      parent_span_id: null,
      name: "UserService.GetByID",
      file: "demo/grpc/internal/server/user.go",
      line: 10,
      start_us: 1,
      duration_us: 1,
      status: "ok",
      entry_kind: "grpc",
      entry_name: "UserService/GetByID",
    };

    expect(grpcMapper.map(span)).toBe("entry:grpc:UserService:GetByID");
  });

  it("maps >=95% of demo/http spans", () => {
    const spans: TraceEvent[] = [
      {
        trace_id: "demo",
        span_id: "root",
        parent_span_id: null,
        name: "HTTP GET /api/users/1",
        file: "demo/http/internal/handler/users.go",
        line: 24,
        start_us: 1,
        duration_us: 100,
        status: "ok",
        entry_kind: "http",
        entry_name: "GET /api/users/1",
      },
      {
        trace_id: "demo",
        span_id: "handler",
        parent_span_id: "root",
        name: "handler.UserHandler.GetByID",
        file: "demo/http/internal/handler/users.go",
        line: 31,
        start_us: 2,
        duration_us: 80,
        status: "ok",
      },
      {
        trace_id: "demo",
        span_id: "service",
        parent_span_id: "handler",
        name: "service.UserService.GetByID",
        file: "demo/http/internal/service/user.go",
        line: 28,
        start_us: 3,
        duration_us: 50,
        status: "ok",
      },
      {
        trace_id: "demo",
        span_id: "repo",
        parent_span_id: "service",
        name: "repository.UserRepository.FindByID",
        file: "demo/http/internal/repository/user.go",
        line: 40,
        start_us: 4,
        duration_us: 20,
        status: "ok",
      },
      {
        trace_id: "demo",
        span_id: "list-root",
        parent_span_id: null,
        name: "HTTP GET /api/items",
        file: "demo/http/internal/handler/items.go",
        line: 20,
        start_us: 5,
        duration_us: 100,
        status: "ok",
        entry_kind: "http",
        entry_name: "GET /api/items/",
      },
      {
        trace_id: "demo",
        span_id: "health",
        parent_span_id: null,
        name: "HTTP GET /health",
        file: "demo/http/main.go",
        line: 1,
        start_us: 6,
        duration_us: 10,
        status: "ok",
        entry_kind: "http",
        entry_name: "GET /health",
      },
      {
        trace_id: "demo",
        span_id: "create",
        parent_span_id: null,
        name: "HTTP POST /api/users",
        file: "demo/http/internal/handler/users.go",
        line: 54,
        start_us: 7,
        duration_us: 120,
        status: "ok",
        entry_kind: "http",
        entry_name: "POST /api/users/",
      },
      {
        trace_id: "demo",
        span_id: "item",
        parent_span_id: "list-root",
        name: "handler.ItemHandler.List",
        file: "demo/http/internal/handler/items.go",
        line: 15,
        start_us: 8,
        duration_us: 30,
        status: "ok",
      },
      {
        trace_id: "demo",
        span_id: "router",
        parent_span_id: "root",
        name: "router.New",
        file: "demo/http/internal/router/router.go",
        line: 20,
        start_us: 9,
        duration_us: 5,
        status: "ok",
      },
      {
        trace_id: "demo",
        span_id: "main",
        parent_span_id: "health",
        name: "main.main",
        file: "demo/http/main.go",
        line: 15,
        start_us: 10,
        duration_us: 5,
        status: "ok",
      },
    ];

    const mapped = spans.map((span) => mapper.map(span));
    const hits = mapped.filter((nodeId) => nodeId !== null).length;
    expect(hits / spans.length).toBeGreaterThanOrEqual(0.95);
  });
});
