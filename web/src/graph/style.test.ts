import { describe, expect, it } from "vitest";
import type { Node } from "@debugviz/protocol";
import { isNodeVisibleAtLod, lodLevelForDistance, nodeColor } from "./style.js";

describe("graph style (issue 3.3)", () => {
  it("colors HTTP entry points by method", () => {
    const getNode: Node = {
      id: "entry:http:GET:/x",
      type: "entry_point",
      kind: "http",
      name: "GET /x",
      metadata: { method: "GET", path: "/x" },
    };
    const postNode: Node = {
      ...getNode,
      id: "entry:http:POST:/x",
      metadata: { method: "POST", path: "/x" },
    };
    expect(nodeColor(getNode)).not.toBe(nodeColor(postNode));
  });

  it("uses distinct colors per entry kind", () => {
    const kinds = ["http", "grpc", "cli", "worker"] as const;
    const colors = kinds.map((kind) =>
      nodeColor({
        id: `entry:${kind}:x`,
        type: "entry_point",
        kind,
        name: kind,
        metadata: {},
      }),
    );
    expect(new Set(colors).size).toBe(4);
  });

  it("applies LOD visibility", () => {
    const pkg: Node = { id: "p", type: "package", name: "p" };
    const fn: Node = { id: "f", type: "function", name: "f" };
    expect(isNodeVisibleAtLod(pkg, lodLevelForDistance(60))).toBe(true);
    expect(isNodeVisibleAtLod(fn, lodLevelForDistance(60))).toBe(false);
    expect(isNodeVisibleAtLod(fn, lodLevelForDistance(10))).toBe(true);
  });
});
