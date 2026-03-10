import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach } from "vitest";
import { useCommandPanes } from "./useCommandPanes";

beforeEach(() => {
  localStorage.clear();
});

describe("useCommandPanes", () => {
  it("initializes with a single leaf pane", () => {
    const { result } = renderHook(() => useCommandPanes());
    expect(result.current.root.kind).toBe("leaf");
    expect(result.current.leafCount).toBe(1);
    expect(result.current.activePaneId).toBe(result.current.root.id);
  });

  it("splits a pane horizontally", () => {
    const { result } = renderHook(() => useCommandPanes());
    const rootId = result.current.root.id;

    act(() => {
      result.current.splitPane(rootId, "horizontal");
    });

    expect(result.current.root.kind).toBe("split");
    expect(result.current.leafCount).toBe(2);
    if (result.current.root.kind === "split") {
      expect(result.current.root.direction).toBe("horizontal");
    }
  });

  it("splits a pane vertically", () => {
    const { result } = renderHook(() => useCommandPanes());
    const rootId = result.current.root.id;

    act(() => {
      result.current.splitPane(rootId, "vertical");
    });

    expect(result.current.root.kind).toBe("split");
    if (result.current.root.kind === "split") {
      expect(result.current.root.direction).toBe("vertical");
    }
  });

  it("closes a pane and collapses parent", () => {
    const { result } = renderHook(() => useCommandPanes());
    const rootId = result.current.root.id;

    act(() => {
      result.current.splitPane(rootId, "horizontal");
    });

    expect(result.current.leafCount).toBe(2);

    // Close the new pane (active one)
    const activePaneId = result.current.activePaneId;
    let lastClosed = false;
    act(() => {
      lastClosed = result.current.closePane(activePaneId);
    });

    expect(lastClosed).toBe(false);
    expect(result.current.leafCount).toBe(1);
    expect(result.current.root.kind).toBe("leaf");
  });

  it("returns true from closePane when last pane", () => {
    const { result } = renderHook(() => useCommandPanes());
    let lastClosed = false;

    act(() => {
      lastClosed = result.current.closePane(result.current.root.id);
    });

    expect(lastClosed).toBe(true);
    expect(result.current.leafCount).toBe(1); // still 1, not removed
  });

  it("sets agent ID on a pane", () => {
    const { result } = renderHook(() => useCommandPanes());
    const paneId = result.current.root.id;

    act(() => {
      result.current.setAgentId(paneId, "agent-123");
    });

    expect(result.current.root.kind).toBe("leaf");
    if (result.current.root.kind === "leaf") {
      expect(result.current.root.agentId).toBe("agent-123");
    }
  });

  it("cycles focus between panes", () => {
    const { result } = renderHook(() => useCommandPanes());
    const rootId = result.current.root.id;

    act(() => {
      result.current.splitPane(rootId, "horizontal");
    });

    const firstActive = result.current.activePaneId;

    act(() => {
      result.current.focusNext();
    });

    expect(result.current.activePaneId).not.toBe(firstActive);

    act(() => {
      result.current.focusPrev();
    });

    expect(result.current.activePaneId).toBe(firstActive);
  });

  it("persists to localStorage", () => {
    const { result } = renderHook(() => useCommandPanes());

    act(() => {
      result.current.setAgentId(result.current.root.id, "agent-456");
    });

    const stored = localStorage.getItem("kf-command-panes");
    expect(stored).not.toBeNull();
    const parsed = JSON.parse(stored!);
    expect(parsed.root.agentId).toBe("agent-456");
  });

  it("resets to initial state", () => {
    const { result } = renderHook(() => useCommandPanes());

    act(() => {
      result.current.splitPane(result.current.root.id, "horizontal");
    });

    expect(result.current.leafCount).toBe(2);

    act(() => {
      result.current.reset();
    });

    expect(result.current.leafCount).toBe(1);
    expect(result.current.root.kind).toBe("leaf");
  });
});
