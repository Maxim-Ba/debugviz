import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { App } from "./App.js";

vi.mock("./hooks/useGraph.js", () => ({
  useGraph: () => ({
    graph: null,
    meta: null,
    error: "404 /api/graph",
    loading: false,
    reload: vi.fn(),
  }),
}));

vi.mock("./hooks/useTraceStream.js", () => ({
  useTraceStream: () => ({
    summaries: [],
    activeTraceId: null,
    spans: [],
    connected: false,
    highlightNodeIds: new Set(),
    errorNodeIds: new Set(),
    selectTrace: vi.fn(),
    refreshSummaries: vi.fn(),
  }),
}));

vi.mock("./scene/GraphViewer.js", () => ({
  GraphViewer: () => <div data-testid="graph-viewer" />,
}));

describe("App", () => {
  it("renders header and graph error state", () => {
    render(<App />);
    expect(screen.getByRole("heading", { name: "DebugViz" })).toBeDefined();
    expect(screen.getByText(/Граф не загружен/)).toBeDefined();
  });
});
