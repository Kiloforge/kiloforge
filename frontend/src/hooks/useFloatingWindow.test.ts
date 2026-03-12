import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useFloatingWindow, detectEdge } from "./useFloatingWindow";

function makePointerEvent(
  overrides: Partial<React.PointerEvent> = {},
): React.PointerEvent {
  return {
    clientX: 0,
    clientY: 0,
    pointerId: 1,
    preventDefault: () => {},
    stopPropagation: () => {},
    target: {
      setPointerCapture: () => {},
      releasePointerCapture: () => {},
    },
    ...overrides,
  } as unknown as React.PointerEvent;
}

describe("useFloatingWindow — zoom compensation", () => {
  let origZoom: string;

  beforeEach(() => {
    origZoom = document.documentElement.style.zoom;
  });

  afterEach(() => {
    document.documentElement.style.zoom = origZoom;
  });

  it("drag delta is divided by zoom factor", () => {
    document.documentElement.style.zoom = "1.5";

    const { result } = renderHook(() =>
      useFloatingWindow({ initialX: 100, initialY: 100 }),
    );

    const startX = result.current.x;
    const startY = result.current.y;

    // Start drag
    act(() => {
      result.current.onDragStart(
        makePointerEvent({ clientX: 300, clientY: 200 }),
      );
    });

    // Move mouse 150px right, 150px down (screen space)
    // At zoom 1.5, logical delta should be 100px each
    act(() => {
      result.current.onDragMove(
        makePointerEvent({ clientX: 450, clientY: 350 }),
      );
    });

    expect(result.current.x).toBe(startX + 100);
    expect(result.current.y).toBe(startY + 100);
  });

  it("resize delta is divided by zoom factor", () => {
    document.documentElement.style.zoom = "1.5";

    const { result } = renderHook(() =>
      useFloatingWindow({
        defaultWidth: 500,
        defaultHeight: 400,
        initialX: 50,
        initialY: 50,
      }),
    );

    const startW = result.current.width;

    // Start resize on east edge
    act(() => {
      result.current.onResizeStart(
        makePointerEvent({ clientX: 550, clientY: 250 }),
        "e",
      );
    });

    // Move mouse 150px right (screen space) → 100px logical at 1.5 zoom
    act(() => {
      result.current.onResizeMove(
        makePointerEvent({ clientX: 700, clientY: 250 }),
      );
    });

    expect(result.current.width).toBe(startW + 100);
  });

  it("no behavior change when zoom is 1.0", () => {
    document.documentElement.style.zoom = "1";

    const { result } = renderHook(() =>
      useFloatingWindow({ initialX: 100, initialY: 100 }),
    );

    const startX = result.current.x;

    act(() => {
      result.current.onDragStart(
        makePointerEvent({ clientX: 300, clientY: 200 }),
      );
    });

    act(() => {
      result.current.onDragMove(
        makePointerEvent({ clientX: 400, clientY: 200 }),
      );
    });

    // 100px mouse movement = 100px logical movement at zoom 1.0
    expect(result.current.x).toBe(startX + 100);
  });

  it("no behavior change when zoom is unset", () => {
    document.documentElement.style.zoom = "";

    const { result } = renderHook(() =>
      useFloatingWindow({ initialX: 100, initialY: 100 }),
    );

    const startX = result.current.x;

    act(() => {
      result.current.onDragStart(
        makePointerEvent({ clientX: 300, clientY: 200 }),
      );
    });

    act(() => {
      result.current.onDragMove(
        makePointerEvent({ clientX: 450, clientY: 200 }),
      );
    });

    expect(result.current.x).toBe(startX + 150);
  });
});

describe("detectEdge — zoom-scaled threshold", () => {
  let origZoom: string;

  beforeEach(() => {
    origZoom = document.documentElement.style.zoom;
  });

  afterEach(() => {
    document.documentElement.style.zoom = origZoom;
  });

  it("uses larger grab zone at higher zoom", () => {
    // At zoom 1.5, EDGE_ZONE (8) × 1.5 = 12px threshold
    document.documentElement.style.zoom = "1.5";

    const rect = { left: 100, right: 500, top: 100, bottom: 400 } as DOMRect;

    // 10px from left edge — within 12px zone at 1.5x, but outside 8px at 1x
    expect(detectEdge(110, 250, rect)).toBe("w");

    // 13px from left edge — outside 12px zone
    expect(detectEdge(113, 250, rect)).toBeNull();
  });

  it("uses standard grab zone at zoom 1.0", () => {
    document.documentElement.style.zoom = "1";

    const rect = { left: 100, right: 500, top: 100, bottom: 400 } as DOMRect;

    // 7px from left edge — within 8px zone
    expect(detectEdge(107, 250, rect)).toBe("w");

    // 9px from left edge — outside 8px zone
    expect(detectEdge(109, 250, rect)).toBeNull();
  });
});
