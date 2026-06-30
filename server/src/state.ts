import type { Graph } from "@debugviz/protocol";
import { GraphSpanMapper, type SpanMapper } from "@debugviz/protocol";

/** In-memory server state shared across route modules. */
export interface AppState {
  graph: Graph | null;
  mapper: SpanMapper | null;
}

export function createAppState(): AppState {
  return { graph: null, mapper: null };
}

export function setGraph(state: AppState, graph: Graph): void {
  state.graph = graph;
  state.mapper = new GraphSpanMapper(graph);
}
