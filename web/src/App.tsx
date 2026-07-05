import { useCallback, useEffect, useMemo, useState } from "react";
import type { GraphMeta, Node } from "@debugviz/protocol";
import { EntryPointPicker, type EntryKindFilter } from "./components/EntryPointPicker.js";
import { TraceTimeline } from "./components/TraceTimeline.js";
import { useGraph } from "./hooks/useGraph.js";
import { useTraceStream } from "./hooks/useTraceStream.js";
import { GraphViewer } from "./scene/GraphViewer.js";

function MetaStats({ meta }: { meta: GraphMeta | null }) {
  if (!meta) {
    return null;
  }
  return (
    <div className="meta-stats">
      <span>{meta.nodes} nodes</span>
      <span>{meta.edges} edges</span>
      <span>{meta.entry_points} entry points</span>
      {meta.packages !== undefined && <span>{meta.packages} packages</span>}
    </div>
  );
}

export function App() {
  const { graph, meta, error, loading } = useGraph();
  const trace = useTraceStream(Boolean(graph));
  const [pickedNode, setPickedNode] = useState<Node | null>(null);
  const [selectedEntryId, setSelectedEntryId] = useState<string | null>(null);
  const [entryKindFilter, setEntryKindFilter] = useState<EntryKindFilter>("all");
  const onNodePick = useCallback((node: Node | null) => setPickedNode(node), []);

  const rootEntryNodeId = useMemo(() => {
    const root = trace.spans.find((span) => !span.parent_span_id && span.node_id);
    return root?.node_id ?? null;
  }, [trace.spans]);

  useEffect(() => {
    if (rootEntryNodeId) {
      setSelectedEntryId(rootEntryNodeId);
    }
  }, [rootEntryNodeId]);

  const handleEntrySelect = useCallback((node: Node) => {
    setSelectedEntryId(node.id);
    setPickedNode(node);
  }, []);

  return (
    <main className="app">
      <header>
        <div>
          <h1>DebugViz</h1>
          <p>3D code graph + live execution path</p>
        </div>
        <MetaStats meta={meta} />
        <div className="status-row">
          <span className={trace.connected ? "status ok" : "status"}>
            WS {trace.connected ? "connected" : "disconnected"}
          </span>
        </div>
      </header>

      <div className="workspace">
        <section className="viewport">
          {loading && <div className="overlay">Загрузка графа…</div>}
          {error && (
            <div className="overlay error">
              <p>Граф не загружен</p>
              <code>{error}</code>
              <p>Загрузите graph.json: POST /api/graph</p>
            </div>
          )}
          {graph && (
            <GraphViewer
              graph={graph}
              highlightNodeIds={trace.highlightNodeIds}
              errorNodeIds={trace.errorNodeIds}
              focusedEntryId={selectedEntryId}
              onNodePick={onNodePick}
            />
          )}
        </section>

        <aside className="sidebar">
          {pickedNode && (
            <div className="node-card">
              <h2>Node</h2>
              <dl>
                <dt>id</dt>
                <dd>{pickedNode.id}</dd>
                <dt>type</dt>
                <dd>{pickedNode.type}</dd>
                <dt>name</dt>
                <dd>{pickedNode.name}</dd>
                {pickedNode.kind && (
                  <>
                    <dt>kind</dt>
                    <dd>{pickedNode.kind}</dd>
                  </>
                )}
              </dl>
            </div>
          )}

          {graph && (
            <EntryPointPicker
              graph={graph}
              selectedId={selectedEntryId}
              kindFilter={entryKindFilter}
              onKindFilterChange={setEntryKindFilter}
              onSelect={handleEntrySelect}
            />
          )}

          <TraceTimeline spans={trace.spans} activeTraceId={trace.activeTraceId} />

          {trace.summaries.length > 0 && (
            <div className="trace-history">
              <h2>Recent traces</h2>
              <ul>
                {trace.summaries.map((item) => (
                  <li key={item.trace_id}>
                    <button type="button" onClick={() => void trace.selectTrace(item.trace_id)}>
                      <span className={item.status === "error" ? "error" : ""}>
                        {item.entry_name ?? item.trace_id}
                      </span>
                      <span className="meta">{item.span_count} spans</span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </aside>
      </div>
    </main>
  );
}
