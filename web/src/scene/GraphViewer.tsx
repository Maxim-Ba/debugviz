import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import type { Graph, Node } from "@debugviz/protocol";
import { useEffect, useRef, useState } from "react";
import { computeForceLayout } from "../graph/layout.js";
import { buildGraphScene, type GraphSceneHandle } from "./buildGraphScene.js";

export interface GraphViewerProps {
  graph: Graph;
  highlightNodeIds?: Set<string>;
  errorNodeIds?: Set<string>;
  onNodePick?: (node: Node | null) => void;
}

export function GraphViewer({
  graph,
  highlightNodeIds = new Set(),
  errorNodeIds = new Set(),
  onNodePick,
}: GraphViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<GraphSceneHandle | null>(null);
  const [lodLabel, setLodLabel] = useState("detail");

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    const layout = computeForceLayout(graph);
    const { scene, camera, renderer, handle } = buildGraphScene(graph, layout.positions, container);
    handleRef.current = handle;

    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;
    controls.maxDistance = 140;
    controls.minDistance = 4;

    const lookAt = {
      x: (layout.bounds.min.x + layout.bounds.max.x) / 2,
      y: (layout.bounds.min.y + layout.bounds.max.y) / 2,
      z: (layout.bounds.min.z + layout.bounds.max.z) / 2,
    };
    controls.target.set(lookAt.x, lookAt.y, lookAt.z);
    controls.maxDistance = Math.max(layout.bounds.max.x - layout.bounds.min.x, 40) * 4;

    let frameId = 0;
    let lastLod = "detail";
    const animate = () => {
      controls.update();
      const dist = camera.position.distanceTo(controls.target);
      handle.setLodDistance(dist);
      let nextLod = "detail";
      if (dist > 55) {
        nextLod = "package";
      } else if (dist > 28) {
        nextLod = "file";
      }
      if (nextLod !== lastLod) {
        lastLod = nextLod;
        setLodLabel(nextLod);
      }
      renderer.render(scene, camera);
      frameId = window.requestAnimationFrame(animate);
    };
    animate();

    const onResize = () => {
      camera.aspect = container.clientWidth / container.clientHeight;
      camera.updateProjectionMatrix();
      renderer.setSize(container.clientWidth, container.clientHeight);
    };
    window.addEventListener("resize", onResize);

    const onClick = (event: MouseEvent) => {
      onNodePick?.(handle.pickNode(event.clientX, event.clientY));
    };
    renderer.domElement.addEventListener("click", onClick);

    return () => {
      window.cancelAnimationFrame(frameId);
      window.removeEventListener("resize", onResize);
      renderer.domElement.removeEventListener("click", onClick);
      controls.dispose();
      handle.dispose();
      handleRef.current = null;
    };
  }, [graph, onNodePick]);

  useEffect(() => {
    handleRef.current?.setHighlights(highlightNodeIds, errorNodeIds);
  }, [highlightNodeIds, errorNodeIds]);

  return (
    <div className="graph-viewer">
      <div ref={containerRef} className="graph-canvas" />
      <div className="lod-badge">LOD: {lodLabel}</div>
    </div>
  );
}
