import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RelationshipOverlay, computeBezierPath } from "./RelationshipOverlay";
import type { TrackDependency, TrackConflict } from "../types/api";

describe("computeBezierPath", () => {
  it("generates a cubic bezier path between two positions", () => {
    const from = { x: 0, y: 50, width: 100, height: 40 };
    const to = { x: 300, y: 80, width: 100, height: 40 };
    const path = computeBezierPath(from, to);
    expect(path).toMatch(/^M .+ C .+, .+, .+$/);
    // Start at center of "from"
    expect(path).toContain("M 50 70");
  });

  it("curves out to the right for same-column (small dx) cards", () => {
    const from = { x: 100, y: 50, width: 100, height: 40 };
    const to = { x: 105, y: 200, width: 100, height: 40 };
    const path = computeBezierPath(from, to);
    // Should use outward curve (control points offset to the right)
    expect(path).toContain("C");
    // Control points should be right of endpoints (x > 155)
    expect(path).toMatch(/C 2\d+/);
  });

  it("applies line spread offset for index > 0", () => {
    const from = { x: 0, y: 50, width: 100, height: 40 };
    const to = { x: 300, y: 80, width: 100, height: 40 };
    const path0 = computeBezierPath(from, to, 0);
    const path1 = computeBezierPath(from, to, 1);
    // Different indices should produce different paths
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
