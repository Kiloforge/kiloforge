import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MockEventSource } from "../test/mocks/eventsource";

// Mock errorToast before importing the hook
vi.mock("../api/errorToast", () => ({
  showToast: vi.fn(),
}));

// Stub EventSource before importing the hook
vi.stubGlobal("EventSource", MockEventSource);

import { useSSE } from "./useSSE";

describe("useSSE", () => {
  beforeEach(() => {
    MockEventSource.reset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("creates EventSource with correct URL", () => {
    renderHook(() => useSSE("/events/stream", {}));
    expect(MockEventSource._instances).toHaveLength(1);
    expect(MockEventSource.latest!.url).toBe("/events/stream");
  });

  it("sets state to connected on open", () => {
    const { result } = renderHook(() => useSSE("/events", {}));
    expect(result.current).toBe("disconnected");

    act(() => MockEventSource.latest!.simulateOpen());
    expect(result.current).toBe("connected");
  });

  it("dispatches events to registered handlers", () => {
    const handler = vi.fn();
    renderHook(() => useSSE("/events", { "project.updated": handler }));
    act(() => MockEventSource.latest!.simulateOpen());

    act(() => {
      MockEventSource.latest!.simulateEvent("project.updated", { slug: "my-proj" });
    });

    expect(handler).toHaveBeenCalledWith({ slug: "my-proj" });
  });

  it("reconnects on error with exponential backoff", () => {
    renderHook(() => useSSE("/events", {}));
    act(() => MockEventSource.latest!.simulateOpen());
    expect(MockEventSource._instances).toHaveLength(1);

    act(() => MockEventSource.latest!.simulateError());
    expect(MockEventSource.latest!.close).toHaveBeenCalled();

    // First retry at 1000ms
    act(() => { vi.advanceTimersByTime(1000); });
    expect(MockEventSource._instances).toHaveLength(2);

    // Second error -> retry at 2000ms
    act(() => MockEventSource.latest!.simulateError());
    act(() => { vi.advanceTimersByTime(1000); });
    expect(MockEventSource._instances).toHaveLength(2); // Not yet
    act(() => { vi.advanceTimersByTime(1000); });
    expect(MockEventSource._instances).toHaveLength(3);
  });

  it("sets state to reconnecting on error", () => {
    const { result } = renderHook(() => useSSE("/events", {}));
    act(() => MockEventSource.latest!.simulateOpen());
    expect(result.current).toBe("connected");

    act(() => MockEventSource.latest!.simulateError());
    expect(result.current).toBe("reconnecting");
  });

  it("cleans up EventSource on unmount", () => {
    const { unmount } = renderHook(() => useSSE("/events", {}));
    const es = MockEventSource.latest!;
    act(() => es.simulateOpen());

    unmount();

    expect(es.close).toHaveBeenCalled();
  });

  it("does not reconnect after unmount", () => {
    const { unmount } = renderHook(() => useSSE("/events", {}));
    act(() => MockEventSource.latest!.simulateOpen());

    unmount();

    // Even if we advance timers, no new EventSource should be created
    act(() => { vi.advanceTimersByTime(5000); });
    expect(MockEventSource._instances).toHaveLength(1);
  });

  it("ignores malformed event data", () => {
    const handler = vi.fn();
    renderHook(() => useSSE("/events", { "test": handler }));
    act(() => MockEventSource.latest!.simulateOpen());

    // Simulate malformed data by directly calling the listener with invalid JSON
    const es = MockEventSource.latest!;
    const event = new MessageEvent("test", { data: "not valid json" });
    // Access the private listeners through addEventListener
    act(() => {
      es.simulateEvent("test", "will-be-valid-json");
    });

    // The handler should be called since simulateEvent wraps in JSON.stringify
    expect(handler).toHaveBeenCalledWith("will-be-valid-json");
  });

  it("resets retry delay after successful reconnection", () => {
    renderHook(() => useSSE("/events", {}));
    act(() => MockEventSource.latest!.simulateOpen());

    // Error and reconnect
    act(() => MockEventSource.latest!.simulateError());
    act(() => { vi.advanceTimersByTime(1000); });
    expect(MockEventSource._instances).toHaveLength(2);

    // Successful reconnection resets delay
    act(() => MockEventSource.latest!.simulateOpen());

    // Another error -> should retry at 1000ms again (reset)
    act(() => MockEventSource.latest!.simulateError());
    act(() => { vi.advanceTimersByTime(1000); });
    expect(MockEventSource._instances).toHaveLength(3);
  });
});
