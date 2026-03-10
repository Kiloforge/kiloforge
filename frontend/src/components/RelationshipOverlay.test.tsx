import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RelationshipOverlay, computeEdgePath } from "./RelationshipOverlay";
import type { TrackDependency, TrackConflict } from "../types/api";

describe("computeEdgePath", () => {
  it("generates a straight-line path between card edges (cross-column)", () => {
    const from = { x: 0, y: 50, width: 100, height: 40 };
    const to = { x: 300, y: 80, width: 100, height: 40 };
    const path = computeEdgePath(from, to);
    // Straight line: M ... L ...
    expect(path).toMatch(/^M .+ L .+$/);
    // from right edge (x=100) to left edge (x=300), vertical center
    expect(path).toBe("M 100 70 L 300 100");
  });

  it("anchors from left/right edges based on direction", () => {
    // to is to the LEFT of from
    const from = { x: 300, y: 50, width: 100, height: 40 };
    const to = { x: 0, y: 80, width: 100, height: 40 };
    const path = computeEdgePath(from, to);
    // from left edge (x=300), to right edge (x=100)
    expect(path).toBe("M 300 70 L 100 100");
  });

  it("uses top/bottom edges for same-column cards", () => {
    const from = { x: 100, y: 50, width: 100, height: 40 };
    const to = { x: 105, y: 200, width: 100, height: 40 };
    const path = computeEdgePath(from, to);
    // Same column (dx < 20): from bottom edge, to top edge
    expect(path).toMatch(/^M .+ L .+$/);
    // from center-x, bottom edge (y=90) → to center-x, top edge (y=200)
    expect(path).toBe("M 150 90 L 155 200");
  });

  it("applies line spread offset for index > 0", () => {
    const from = { x: 0, y: 50, width: 100, height: 40 };
    const to = { x: 300, y: 80, width: 100, height: 40 };
    const path0 = computeEdgePath(from, to, 0);
    const path1 = computeEdgePath(from, to, 1);
    // Different indices should produce different paths (spread offset)
    expect(path0).not.toBe(path1);
  });
});

describe("RelationshipOverlay", () => {
  function setup(
    deps: Map<string, TrackDependency[]> = new Map(),
    conflicts: Map<string, TrackConflict[]> = new Map(),
    visible = true,
  ) {
    const containerRef = { current: document.createElement("div") };
    const cardRefs = new Map<string, HTMLElement>();
    const onToggle = vi.fn();

    const result = render(
      <RelationshipOverlay
        cardRefs={cardRefs}
        containerRef={containerRef}
        dependencies={deps}
        conflicts={conflicts}
        visible={visible}
        onToggle={onToggle}
      />,
    );

    return { ...result, onToggle };
  }

  it("renders nothing when no deps or conflicts", () => {
    const { container } = setup();
    expect(container.innerHTML).toBe("");
  });

  it("renders toggle when deps data exists", () => {
    const deps = new Map([
      ["track-a", [{ id: "track-b", status: "complete" }]],
    ]);
    setup(deps);
    expect(screen.getByText("Hide relations")).toBeInTheDocument();
  });

  it("shows 'Show relations' when visible is false", () => {
    const deps = new Map([
      ["track-a", [{ id: "track-b", status: "complete" }]],
    ]);
    setup(deps, new Map(), false);
    expect(screen.getByText("Show relations")).toBeInTheDocument();
  });

  it("calls onToggle when toggle is clicked", async () => {
    const user = userEvent.setup();
    const deps = new Map([
      ["track-a", [{ id: "track-b", status: "complete" }]],
    ]);
    const { onToggle } = setup(deps);
    await user.click(screen.getByText("Hide relations"));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it("renders toggle when conflicts data exists", () => {
    const conflicts = new Map([
      ["track-a", [{ track_id: "track-b", risk: "high" as const }]],
    ]);
    setup(new Map(), conflicts);
    expect(screen.getByText("Hide relations")).toBeInTheDocument();
  });
});
