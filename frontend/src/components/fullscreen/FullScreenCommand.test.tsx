import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, act } from "@testing-library/react";
import { FullScreenCommand } from "./FullScreenCommand";

// Mock hooks used by FullScreenCommand
vi.mock("../../hooks/useCommandPanes", () => ({
  useCommandPanes: () => ({
    root: { kind: "leaf" as const, id: "pane-1", agentId: null },
    leafCount: 1,
    activePaneId: "pane-1",
    setActivePaneId: vi.fn(),
    splitPane: vi.fn(),
    closePane: vi.fn(),
    focusNext: vi.fn(),
    focusPrev: vi.fn(),
    setAgentId: vi.fn(),
  }),
}));

vi.mock("../../hooks/usePlatform", () => ({
  usePlatform: () => ({ mod: "⌘" }),
}));

vi.mock("./SplitContainer", () => ({
  SplitContainer: () => <div data-testid="split-container" />,
}));

vi.mock("./CommandModeHelp", () => ({
  CommandModeHelp: () => <div data-testid="help" />,
}));

describe("FullScreenCommand fullscreen API", () => {
  let requestFullscreenMock: ReturnType<typeof vi.fn>;
  let exitFullscreenMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    requestFullscreenMock = vi.fn().mockResolvedValue(undefined);
    exitFullscreenMock = vi.fn().mockResolvedValue(undefined);

    // Set up document.documentElement.requestFullscreen
    Object.defineProperty(document.documentElement, "requestFullscreen", {
      value: requestFullscreenMock,
      writable: true,
      configurable: true,
    });

    Object.defineProperty(document, "exitFullscreen", {
      value: exitFullscreenMock,
      writable: true,
      configurable: true,
    });

    Object.defineProperty(document, "fullscreenElement", {
      value: document.documentElement,
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls requestFullscreen on mount", () => {
    render(<FullScreenCommand agents={[]} onExit={vi.fn()} />);
    expect(requestFullscreenMock).toHaveBeenCalledTimes(1);
  });

  it("calls exitFullscreen on unmount when in fullscreen", () => {
    const { unmount } = render(
      <FullScreenCommand agents={[]} onExit={vi.fn()} />,
    );
    unmount();
    expect(exitFullscreenMock).toHaveBeenCalledTimes(1);
  });

  it("does not call exitFullscreen on unmount when not in fullscreen", () => {
    Object.defineProperty(document, "fullscreenElement", {
      value: null,
      writable: true,
      configurable: true,
    });

    const { unmount } = render(
      <FullScreenCommand agents={[]} onExit={vi.fn()} />,
    );
    unmount();
    expect(exitFullscreenMock).not.toHaveBeenCalled();
  });

  it("calls onExit when fullscreenchange fires and fullscreen is exited", () => {
    Object.defineProperty(document, "fullscreenElement", {
      value: document.documentElement,
      writable: true,
      configurable: true,
    });

    const onExit = vi.fn();
    render(<FullScreenCommand agents={[]} onExit={onExit} />);

    // Simulate browser exiting fullscreen
    Object.defineProperty(document, "fullscreenElement", {
      value: null,
      writable: true,
      configurable: true,
    });

    act(() => {
      document.dispatchEvent(new Event("fullscreenchange"));
    });

    expect(onExit).toHaveBeenCalledTimes(1);
  });

  it("does not call onExit when fullscreenchange fires but still in fullscreen", () => {
    const onExit = vi.fn();
    render(<FullScreenCommand agents={[]} onExit={onExit} />);

    // fullscreenElement is still set (entering fullscreen triggers this event too)
    act(() => {
      document.dispatchEvent(new Event("fullscreenchange"));
    });

    expect(onExit).not.toHaveBeenCalled();
  });

  it("handles requestFullscreen rejection gracefully", () => {
    requestFullscreenMock.mockRejectedValue(new Error("Not allowed"));

    // Should not throw
    expect(() => {
      render(<FullScreenCommand agents={[]} onExit={vi.fn()} />);
    }).not.toThrow();
  });

  it("handles missing requestFullscreen gracefully", () => {
    Object.defineProperty(document.documentElement, "requestFullscreen", {
      value: undefined,
      writable: true,
      configurable: true,
    });

    // Should not throw
    expect(() => {
      render(<FullScreenCommand agents={[]} onExit={vi.fn()} />);
    }).not.toThrow();
  });
});
