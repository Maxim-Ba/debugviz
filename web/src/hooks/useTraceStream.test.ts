import { describe, expect, it } from "vitest";
import type { StoredSpan } from "../api/client.js";

function buildPathNodeIds(spans: StoredSpan[]): { highlights: Set<string>; errors: Set<string> } {
  const highlights = new Set<string>();
  const errors = new Set<string>();
  for (const span of spans) {
    if (span.node_id) {
      highlights.add(span.node_id);
      if (span.status === "error") {
        errors.add(span.node_id);
      }
    }
  }
  return { highlights, errors };
}

describe("trace path highlights (issue 3.4)", () => {
  it("collects node ids and error spans", () => {
    const spans: StoredSpan[] = [
      {
        trace_id: "t1",
        span_id: "s1",
        name: "root",
        file: "a.go",
        line: 1,
        start_us: 1,
        duration_us: 10,
        status: "ok",
        node_id: "entry:http:GET:/x",
      },
      {
        trace_id: "t1",
        span_id: "s2",
        parent_span_id: "s1",
        name: "inner",
        file: "b.go",
        line: 2,
        start_us: 2,
        duration_us: 5,
        status: "error",
        node_id: "func:b.go:Fail",
      },
    ];

    const path = buildPathNodeIds(spans);
    expect(path.highlights.has("entry:http:GET:/x")).toBe(true);
    expect(path.highlights.has("func:b.go:Fail")).toBe(true);
    expect(path.errors.has("func:b.go:Fail")).toBe(true);
    expect(path.errors.has("entry:http:GET:/x")).toBe(false);
  });
});
