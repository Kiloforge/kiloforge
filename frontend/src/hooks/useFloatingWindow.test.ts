import { describe, it, expect, afterEach } from "vitest";
import { detectEdge, getZoom } from "./useFloatingWindow";

afterEach(() => {
  document.documentElement.style.zoom = "";
});

function makeRect(x: number, y: number, w: number, h: number): DOMRect {
  return new DOMRect(x, y, w, h);
}

describe("getZoom", () => {
  it("returns 1 when zoom is unset", () => {
    document.documentElement.style.zoom = "";
    expect(getZoom()).toBe(1);
  });

  it("returns the numeric zoom value", () => {
    document.documentElement.style.zoom = "1.5";
    expect(getZoom()).toBe(1.5);
  });

  it("returns 1 for invalid zoom values", () => {
    document.documentElement.style.zoom = "invalid";
    expect(getZoom()).toBe(1);
  });
});

describe("detectEdge with zoom compensation", () => {
  const rect = makeRect(100, 100, 200, 200); // left=100, top=100, right=300, bottom=300

  it("detects left edge at 100% zoom", () => {
    document.documentElement.style.zoom = "1";
    // 5px from left edge, within 8px zone
    expect(detectEdge(105, 200, rect)).toBe("w");
  });

  it("does not detect edge outside zone at 100% zoom", () => {
    document.documentElement.style.zoom = "1";
    // 10px from left edge, outside 8px zone
    expect(detectEdge(110, 200, rect)).toBeNull();
  });

  it("expands edge zone at 150% zoom", () => {
    document.documentElement.style.zoom = "1.5";
    // 10px from left edge — outside 8px but inside 12px (8*1.5)
    expect(detectEdge(110, 200, rect)).toBe("w");
  });

  it("shrinks edge zone at 75% zoom", () => {
    document.documentElement.style.zoom = "0.75";
    // 7px from left edge — inside 8px but outside 6px (8*0.75)
    expect(detectEdge(107, 200, rect)).toBeNull();
  });

  it("detects corner edges with zoom", () => {
    document.documentElement.style.zoom = "1.5";
    // 10px from both top-left edges, within 12px zone
    expect(detectEdge(110, 110, rect)).toBe("nw");
  });

  it("no behavior change at zoom=1.0", () => {
    document.documentElement.style.zoom = "1";
    // Exactly at boundary: 8px from left, 8px from top — NOT inside (< not <=)
    expect(detectEdge(108, 108, rect)).toBeNull();
    // 7px from left and top — inside
    expect(detectEdge(107, 107, rect)).toBe("nw");
  });
});
