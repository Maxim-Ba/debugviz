import { readFile } from "node:fs/promises";
import type { AppState } from "./state.js";
import { setGraph } from "./state.js";
import { assertValidGraph } from "./validate.js";

/** Load a static graph from INIT_GRAPH_PATH when the server starts (Docker one-liner). */
export async function loadInitialGraph(state: AppState): Promise<void> {
  const path = process.env.INIT_GRAPH_PATH?.trim();
  if (!path) {
    return;
  }

  const raw = await readFile(path, "utf8");
  const graph = assertValidGraph(JSON.parse(raw));
  setGraph(state, graph);
}
