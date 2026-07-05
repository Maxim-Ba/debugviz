import type { EntryKind, Graph, Node } from "@debugviz/protocol";
import {
  ENTRY_KIND_LABELS,
  ENTRY_KINDS,
  entryKindCounts,
  entryPointSubtitle,
  listEntryPoints,
} from "../graph/entryPoints.js";

export type EntryKindFilter = EntryKind | "all";

export interface EntryPointPickerProps {
  graph: Graph;
  selectedId: string | null;
  kindFilter: EntryKindFilter;
  onKindFilterChange: (kind: EntryKindFilter) => void;
  onSelect: (node: Node) => void;
}

export function EntryPointPicker({
  graph,
  selectedId,
  kindFilter,
  onKindFilterChange,
  onSelect,
}: EntryPointPickerProps) {
  const counts = entryKindCounts(graph);
  const total = listEntryPoints(graph).length;
  const visible = listEntryPoints(graph, kindFilter === "all" ? undefined : kindFilter);

  return (
    <div className="entry-picker">
      <h2>Entry points</h2>
      <div className="kind-tabs" role="tablist" aria-label="Filter by entry kind">
        <button
          type="button"
          role="tab"
          className={kindFilter === "all" ? "active" : ""}
          aria-selected={kindFilter === "all"}
          onClick={() => onKindFilterChange("all")}
        >
          All <span className="badge">{total}</span>
        </button>
        {ENTRY_KINDS.map((kind) => (
          <button
            key={kind}
            type="button"
            role="tab"
            className={kindFilter === kind ? "active" : ""}
            aria-selected={kindFilter === kind}
            onClick={() => onKindFilterChange(kind)}
          >
            {ENTRY_KIND_LABELS[kind]} <span className="badge">{counts[kind]}</span>
          </button>
        ))}
      </div>
      <ul className="entry-list">
        {visible.length === 0 && <li className="entry-list-empty">No entry points</li>}
        {visible.map((node) => (
          <li key={node.id}>
            <button
              type="button"
              className={`entry-list-item${selectedId === node.id ? " selected" : ""}`}
              onClick={() => onSelect(node)}
            >
              <span className={`entry-kind-badge kind-${node.kind}`}>{node.kind}</span>
              <span className="entry-list-text">
                <span className="entry-name">{node.name}</span>
                <span className="entry-subtitle">{entryPointSubtitle(node)}</span>
              </span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
