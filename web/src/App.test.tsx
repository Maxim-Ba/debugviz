import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { App } from "./App.js";

vi.mock("three", () => ({
  Scene: vi.fn(() => ({ background: null, add: vi.fn() })),
  Color: vi.fn(),
  PerspectiveCamera: vi.fn(() => ({
    position: { set: vi.fn() },
    aspect: 1,
    updateProjectionMatrix: vi.fn(),
  })),
  WebGLRenderer: vi.fn(() => ({
    setPixelRatio: vi.fn(),
    setSize: vi.fn(),
    domElement: document.createElement("canvas"),
    render: vi.fn(),
    dispose: vi.fn(),
  })),
  BoxGeometry: vi.fn(() => ({ dispose: vi.fn() })),
  MeshStandardMaterial: vi.fn(() => ({ dispose: vi.fn() })),
  Mesh: vi.fn(() => ({ rotation: { x: 0, y: 0 } })),
  DirectionalLight: vi.fn(() => ({ position: { set: vi.fn() } })),
}));

describe("App", () => {
  it("renders bootstrap heading", () => {
    render(<App />);
    expect(screen.getByRole("heading", { name: "DebugViz" })).toBeDefined();
  });
});
