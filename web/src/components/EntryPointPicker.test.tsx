import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Graph } from "@debugviz/protocol";
import { EntryPointPicker } from "./EntryPointPicker.js";

const graph: Graph = {
  version: "2",
  nodes: [
    { id: "entry:http:GET:/health", type: "entry_point", kind: "http", name: "GET /health" },
    { id: "entry:grpc:svc:Method", type: "entry_point", kind: "grpc", name: "svc/Method" },
    { id: "entry:cli:migrate-up", type: "entry_point", kind: "cli", name: "migrate up" },
  ],
  edges: [],
};

describe("EntryPointPicker", () => {
  it("renders kind tabs and entry list", () => {
    render(
      <EntryPointPicker
        graph={graph}
        selectedId={null}
        kindFilter="all"
        onKindFilterChange={vi.fn()}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByRole("heading", { name: "Entry points" })).toBeDefined();
    expect(screen.getByText("GET /health")).toBeDefined();
    expect(screen.getByText("svc/Method")).toBeDefined();
  });

  it("filters entries by kind tab", () => {
    const onKindFilterChange = vi.fn();
    render(
      <EntryPointPicker
        graph={graph}
        selectedId={null}
        kindFilter="grpc"
        onKindFilterChange={onKindFilterChange}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText("svc/Method")).toBeDefined();
    expect(screen.queryByText("GET /health")).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: /HTTP/i }));
    expect(onKindFilterChange).toHaveBeenCalledWith("http");
  });

  it("calls onSelect when entry clicked", () => {
    const onSelect = vi.fn();
    render(
      <EntryPointPicker
        graph={graph}
        selectedId={null}
        kindFilter="all"
        onKindFilterChange={vi.fn()}
        onSelect={onSelect}
      />,
    );
    fireEvent.click(screen.getByText("migrate up"));
    expect(onSelect).toHaveBeenCalledWith(graph.nodes[2]);
  });
});
