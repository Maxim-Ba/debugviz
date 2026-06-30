import type { Graph, Node } from "@debugviz/protocol";

export interface Vec3 {
  x: number;
  y: number;
  z: number;
}

interface SimNode extends Vec3 {
  vx: number;
  vy: number;
  vz: number;
}

export interface LayoutResult {
  positions: Map<string, Vec3>;
  bounds: { min: Vec3; max: Vec3 };
}

function nodeWeight(node: Node): number {
  switch (node.type) {
    case "package":
      return 2.4;
    case "file":
      return 1.6;
    case "entry_point":
      return 2;
    case "middleware":
      return 1.2;
    default:
      return 1;
  }
}

/**
 * Simple 3D force-directed layout (issue 3.2A).
 * O(n²) repulsion — acceptable up to ~500 nodes for MVP.
 */
export function computeForceLayout(
  graph: Graph,
  iterations = 80,
  seed = 42,
): LayoutResult {
  const rng = mulberry32(seed);
  const sim = new Map<string, SimNode>();

  for (const node of graph.nodes) {
    sim.set(node.id, {
      x: (rng() - 0.5) * 20,
      y: (rng() - 0.5) * 20,
      z: (rng() - 0.5) * 20,
      vx: 0,
      vy: 0,
      vz: 0,
    });
  }

  const nodeById = new Map(graph.nodes.map((n) => [n.id, n]));
  const repulsion = 8;
  const attraction = 0.04;
  const damping = 0.85;

  for (let iter = 0; iter < iterations; iter++) {
    const cooling = 1 - iter / iterations;
    const ids = [...sim.keys()];

    for (let i = 0; i < ids.length; i++) {
      const a = sim.get(ids[i]!)!;
      const aNode = nodeById.get(ids[i]!);
      const aWeight = aNode ? nodeWeight(aNode) : 1;

      for (let j = i + 1; j < ids.length; j++) {
        const b = sim.get(ids[j]!)!;
        const dx = a.x - b.x;
        const dy = a.y - b.y;
        const dz = a.z - b.z;
        const distSq = dx * dx + dy * dy + dz * dz + 0.01;
        const dist = Math.sqrt(distSq);
        const force = (repulsion * aWeight * cooling) / distSq;
        const fx = (dx / dist) * force;
        const fy = (dy / dist) * force;
        const fz = (dz / dist) * force;
        a.vx += fx;
        a.vy += fy;
        a.vz += fz;
        b.vx -= fx;
        b.vy -= fy;
        b.vz -= fz;
      }
    }

    for (const edge of graph.edges) {
      const a = sim.get(edge.source);
      const b = sim.get(edge.target);
      if (!a || !b) {
        continue;
      }
      const dx = b.x - a.x;
      const dy = b.y - a.y;
      const dz = b.z - a.z;
      const dist = Math.sqrt(dx * dx + dy * dy + dz * dz) + 0.01;
      const strength =
        edge.type === "calls" || edge.type === "entry_handles" ? attraction * 2 : attraction;
      const fx = (dx / dist) * strength * dist;
      const fy = (dy / dist) * strength * dist;
      const fz = (dz / dist) * strength * dist;
      a.vx += fx;
      a.vy += fy;
      a.vz += fz;
      b.vx -= fx;
      b.vy -= fy;
      b.vz -= fz;
    }

    for (const node of sim.values()) {
      node.vx *= damping;
      node.vy *= damping;
      node.vz *= damping;
      node.x += node.vx;
      node.y += node.vy;
      node.z += node.vz;
    }
  }

  const positions = new Map<string, Vec3>();
  const min: Vec3 = { x: Infinity, y: Infinity, z: Infinity };
  const max: Vec3 = { x: -Infinity, y: -Infinity, z: -Infinity };

  for (const [id, node] of sim) {
    positions.set(id, { x: node.x, y: node.y, z: node.z });
    min.x = Math.min(min.x, node.x);
    min.y = Math.min(min.y, node.y);
    min.z = Math.min(min.z, node.z);
    max.x = Math.max(max.x, node.x);
    max.y = Math.max(max.y, node.y);
    max.z = Math.max(max.z, node.z);
  }

  return { positions, bounds: { min, max } };
}

function mulberry32(seed: number): () => number {
  let t = seed;
  return () => {
    t += 0x6d2b79f5;
    let r = Math.imul(t ^ (t >>> 15), 1 | t);
    r ^= r + Math.imul(r ^ (r >>> 7), 61 | r);
    return ((r ^ (r >>> 14)) >>> 0) / 4294967296;
  };
}

/** Generate a synthetic graph for perf tests (issue 3.1: 500 nodes). */
export function syntheticGraph(nodeCount: number): Graph {
  const nodes: Graph["nodes"] = [];
  const edges: Graph["edges"] = [];

  for (let i = 0; i < nodeCount; i++) {
    const type =
      i % 17 === 0
        ? "package"
        : i % 5 === 0
          ? "file"
          : i % 23 === 0
            ? "entry_point"
            : "function";
    nodes.push({
      id: `node:${i}`,
      type,
      name: `node_${i}`,
      ...(type === "entry_point"
        ? { kind: "http" as const, metadata: { method: "GET", path: `/api/${i}` } }
        : {}),
    });
    if (i > 0) {
      edges.push({
        type: i % 3 === 0 ? "calls" : "imports",
        source: `node:${i - 1}`,
        target: `node:${i}`,
      });
    }
  }

  return { version: "2", nodes, edges };
}
