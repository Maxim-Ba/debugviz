import type { Graph, Node } from "@debugviz/protocol";
import * as THREE from "three";
import type { Vec3 } from "../graph/layout.js";
import {
  DIM_COLOR,
  ERROR_COLOR,
  HIGHLIGHT_COLOR,
  edgeColor,
  edgeWidth,
  isNodeVisibleAtLod,
  lodLevelForDistance,
  nodeColor,
  nodeGeometryKind,
  nodeScale,
  type LodLevel,
} from "../graph/style.js";

export interface GraphSceneHandle {
  setLodDistance(distance: number): void;
  setHighlights(nodeIds: Set<string>, errorNodeIds?: Set<string>): void;
  pickNode(clientX: number, clientY: number): Node | null;
  dispose(): void;
}

interface InstanceGroup {
  mesh: THREE.InstancedMesh;
  nodeIds: Node[];
}

export function buildGraphScene(
  graph: Graph,
  positions: Map<string, Vec3>,
  container: HTMLElement,
): {
  scene: THREE.Scene;
  camera: THREE.PerspectiveCamera;
  renderer: THREE.WebGLRenderer;
  handle: GraphSceneHandle;
} {
  const scene = new THREE.Scene();
  scene.background = new THREE.Color(0x0f172a);

  const bounds = computeBounds(positions);
  const center = bounds.center;
  const radius = Math.max(bounds.radius, 6);
  const fogFar = Math.max(radius * 8, 80);
  scene.fog = new THREE.Fog(0x0f172a, fogFar * 0.4, fogFar);

  const camera = new THREE.PerspectiveCamera(
    55,
    Math.max(container.clientWidth, 1) / Math.max(container.clientHeight, 1),
    0.1,
    fogFar * 2,
  );
  const cameraDistance = radius * 2.8;
  camera.position.set(
    center.x + cameraDistance * 0.55,
    center.y + cameraDistance * 0.45,
    center.z + cameraDistance * 0.75,
  );
  camera.lookAt(center.x, center.y, center.z);

  const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
  renderer.setSize(container.clientWidth, container.clientHeight);
  container.appendChild(renderer.domElement);

  scene.add(new THREE.AmbientLight(0xffffff, 0.6));

  const geometries = {
    sphere: new THREE.SphereGeometry(1, 12, 10),
    box: new THREE.BoxGeometry(1, 1, 1),
    octahedron: new THREE.OctahedronGeometry(1, 0),
  };

  const groups = new Map<string, InstanceGroup>();
  const buckets = new Map<string, { nodes: Node[] }>();

  for (const node of graph.nodes) {
    const kind = nodeGeometryKind(node);
    const bucket = buckets.get(kind) ?? { nodes: [] };
    bucket.nodes.push(node);
    buckets.set(kind, bucket);
  }

  const baseColors = new Map<string, number>();
  const visibility = new Map<string, boolean>();
  let currentLod: LodLevel = "detail";

  for (const [kind, bucket] of buckets) {
    const geometry = geometries[kind as keyof typeof geometries];
    const material = new THREE.MeshBasicMaterial({ color: 0xffffff });
    const mesh = new THREE.InstancedMesh(geometry, material, bucket.nodes.length);
    mesh.frustumCulled = false;
    mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
    mesh.instanceColor = new THREE.InstancedBufferAttribute(
      new Float32Array(bucket.nodes.length * 3),
      3,
    );

    const nodeIds: Node[] = [];
    const matrix = new THREE.Matrix4();
    const color = new THREE.Color();
    const scale = new THREE.Vector3();

    bucket.nodes.forEach((node, index) => {
      const pos = positions.get(node.id) ?? { x: 0, y: 0, z: 0 };
      const s = nodeScale(node);
      scale.set(s, s, s);
      matrix.compose(
        new THREE.Vector3(pos.x, pos.y, pos.z),
        new THREE.Quaternion(),
        scale,
      );
      mesh.setMatrixAt(index, matrix);
      const c = nodeColor(node);
      baseColors.set(node.id, c);
      color.setHex(c);
      mesh.setColorAt(index, color);
      nodeIds.push(node);
      visibility.set(node.id, true);
    });

    mesh.instanceMatrix.needsUpdate = true;
    if (mesh.instanceColor) {
      mesh.instanceColor.needsUpdate = true;
    }
    scene.add(mesh);
    groups.set(kind, { mesh, nodeIds });
  }

  const edgeGroup = buildEdges(graph, positions);
  scene.add(edgeGroup.lines);

  const highlightNodes = new Set<string>();
  const errorNodes = new Set<string>();
  const raycaster = new THREE.Raycaster();
  const pointer = new THREE.Vector2();

  function updateInstanceMatrices(): void {
    for (const group of groups.values()) {
      const matrix = new THREE.Matrix4();
      const scale = new THREE.Vector3();
      const color = new THREE.Color();

      group.nodeIds.forEach((node, index) => {
        const pos = positions.get(node.id) ?? { x: 0, y: 0, z: 0 };
        const visible = visibility.get(node.id) ?? true;
        const s = visible ? nodeScale(node) : 0.001;
        scale.set(s, s, s);
        matrix.compose(new THREE.Vector3(pos.x, pos.y, pos.z), new THREE.Quaternion(), scale);
        group.mesh.setMatrixAt(index, matrix);

        let hex = baseColors.get(node.id) ?? 0x94a3b8;
        if (!visible) {
          hex = DIM_COLOR;
        } else if (errorNodes.has(node.id)) {
          hex = ERROR_COLOR;
        } else if (highlightNodes.has(node.id)) {
          hex = HIGHLIGHT_COLOR;
        }
        color.setHex(hex);
        group.mesh.setColorAt(index, color);
      });

      group.mesh.instanceMatrix.needsUpdate = true;
      if (group.mesh.instanceColor) {
        group.mesh.instanceColor.needsUpdate = true;
      }
    }
  }

  function applyLod(distance: number): void {
    const lod = lodLevelForDistance(distance);
    if (lod === currentLod) {
      return;
    }
    currentLod = lod;
    for (const node of graph.nodes) {
      visibility.set(node.id, isNodeVisibleAtLod(node, lod));
    }
    updateInstanceMatrices();
  }

  function setHighlights(nodeIds: Set<string>, errorNodeIds: Set<string> = new Set()): void {
    highlightNodes.clear();
    errorNodes.clear();
    for (const id of nodeIds) {
      highlightNodes.add(id);
    }
    for (const id of errorNodeIds) {
      errorNodes.add(id);
    }
    updateInstanceMatrices();
  }

  function pickNode(clientX: number, clientY: number): Node | null {
    const rect = renderer.domElement.getBoundingClientRect();
    pointer.x = ((clientX - rect.left) / rect.width) * 2 - 1;
    pointer.y = -((clientY - rect.top) / rect.height) * 2 + 1;
    raycaster.setFromCamera(pointer, camera);
    const meshes = [...groups.values()].map((g) => g.mesh);
    const hits = raycaster.intersectObjects(meshes, false);
    if (!hits.length) {
      return null;
    }
    const hit = hits[0]!;
    const mesh = hit.object as THREE.InstancedMesh;
    const group = [...groups.values()].find((g) => g.mesh === mesh);
    if (!group || hit.instanceId === undefined) {
      return null;
    }
    return group.nodeIds[hit.instanceId] ?? null;
  }

  const handle: GraphSceneHandle = {
    setLodDistance: applyLod,
    setHighlights,
    pickNode,
    dispose: () => {
      for (const group of groups.values()) {
        group.mesh.geometry.dispose();
        (group.mesh.material as THREE.Material).dispose();
      }
      edgeGroup.dispose();
      for (const geometry of Object.values(geometries)) {
        geometry.dispose();
      }
      renderer.dispose();
      container.removeChild(renderer.domElement);
    },
  };

  const initialDistance = camera.position.distanceTo(
    new THREE.Vector3(center.x, center.y, center.z),
  );
  handle.setLodDistance(initialDistance);

  return { scene, camera, renderer, handle };
}

function buildEdges(
  graph: Graph,
  positions: Map<string, Vec3>,
): { lines: THREE.Group; dispose: () => void } {
  const group = new THREE.Group();
  const byType = new Map<string, { positions: number[]; colors: number[] }>();

  for (const edge of graph.edges) {
    const a = positions.get(edge.source);
    const b = positions.get(edge.target);
    if (!a || !b) {
      continue;
    }
    const bucket = byType.get(edge.type) ?? { positions: [], colors: [] };
    const color = new THREE.Color(edgeColor(edge));
    bucket.positions.push(a.x, a.y, a.z, b.x, b.y, b.z);
    bucket.colors.push(color.r, color.g, color.b, color.r, color.g, color.b);
    byType.set(edge.type, bucket);
  }

  const disposables: THREE.BufferGeometry[] = [];
  const materials: THREE.LineBasicMaterial[] = [];

  for (const [type, bucket] of byType) {
    const edge = graph.edges.find((e) => e.type === type);
    const width = edge ? edgeWidth(edge) : 1;
    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute("position", new THREE.Float32BufferAttribute(bucket.positions, 3));
    geometry.setAttribute("color", new THREE.Float32BufferAttribute(bucket.colors, 3));
    const material = new THREE.LineBasicMaterial({
      vertexColors: true,
      transparent: true,
      opacity: type === "imports" ? 0.35 : 0.75,
      linewidth: width,
    });
    const lines = new THREE.LineSegments(geometry, material);
    lines.frustumCulled = false;
    group.add(lines);
    disposables.push(geometry);
    materials.push(material);
  }

  return {
    lines: group,
    dispose: () => {
      for (const g of disposables) {
        g.dispose();
      }
      for (const m of materials) {
        m.dispose();
      }
    },
  };
}

function computeBounds(positions: Map<string, Vec3>): {
  center: Vec3;
  radius: number;
} {
  let minX = Infinity;
  let minY = Infinity;
  let minZ = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  let maxZ = -Infinity;

  for (const p of positions.values()) {
    minX = Math.min(minX, p.x);
    minY = Math.min(minY, p.y);
    minZ = Math.min(minZ, p.z);
    maxX = Math.max(maxX, p.x);
    maxY = Math.max(maxY, p.y);
    maxZ = Math.max(maxZ, p.z);
  }

  const center = {
    x: (minX + maxX) / 2,
    y: (minY + maxY) / 2,
    z: (minZ + maxZ) / 2,
  };
  const radius = Math.max(maxX - minX, maxY - minY, maxZ - minZ, 8) / 2;
  return { center, radius };
}
