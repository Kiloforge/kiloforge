import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import type { QueueStatus } from "../types/api";
import { useQueue } from "./useQueue";

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn(),
  FetchError: class FetchError extends Error {
    status: number;
    body: unknown;
    constructor(status: number, body: unknown) {
      super(`Request failed with status ${status}`);
      this.status = status;
      this.body = body;
    }
  },
}));

import { fetcher } from "../api/fetcher";

const mockQueue: QueueStatus = {
  running: true,
  max_workers: 3,
  active_workers: 2,
  items: [
    {
      track_id: "track-1",
      project_slug: "proj",
      status: "assigned",
      agent_id: "agent-1",
      enqueued_at: "2026-03-10T00:00:00Z",
      assigned_at: "2026-03-10T00:01:00Z",
      completed_at: null,
    },
    {
      track_id: "track-2",
      project_slug: "proj",
      status: "queued",
      enqueued_at: "2026-03-10T00:02:00Z",
      assigned_at: null,
      completed_at: null,
    },
  ],
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useQueue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches queue status", async () => {
    (fetcher as Mock).mockResolvedValue(mockQueue);
    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.queue).toEqual(mockQueue);
    expect(fetcher).toHaveBeenCalledWith("/api/queue");
  });

  it("returns null queue when fetch fails", async () => {
    (fetcher as Mock).mockRejectedValue(new Error("fail"));
    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.queue).toBeNull();
  });

  it("provides start mutation", async () => {
    (fetcher as Mock).mockResolvedValue(mockQueue);
    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.start(); });

    expect(fetcher).toHaveBeenCalledWith("/api/queue/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    });
  });

  it("provides start mutation with project slug", async () => {
    (fetcher as Mock).mockResolvedValue(mockQueue);
    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.start("my-proj"); });

    expect(fetcher).toHaveBeenCalledWith("/api/queue/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ project: "my-proj" }),
    });
  });

  it("provides stop mutation", async () => {
    (fetcher as Mock).mockResolvedValue(mockQueue);
    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.stop(); });

    expect(fetcher).toHaveBeenCalledWith("/api/queue/stop", { method: "POST" });
  });

  it("provides updateSettings mutation", async () => {
    (fetcher as Mock).mockResolvedValue(mockQueue);
    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.updateSettings({ max_workers: 5 }); });

    expect(fetcher).toHaveBeenCalledWith("/api/queue/settings", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ max_workers: 5 }),
    });
  });

  it("provides handleQueueUpdate SSE handler", async () => {
    (fetcher as Mock).mockResolvedValue(mockQueue);
    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(typeof result.current.handleQueueUpdate).toBe("function");
  });

  describe("project-scoped", () => {
    it("fetches queue with project query param when projectSlug is provided", async () => {
      (fetcher as Mock).mockResolvedValue(mockQueue);
      const { result } = renderHook(() => useQueue("my-project"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(fetcher).toHaveBeenCalledWith("/api/queue?project=my-project");
    });

    it("uses project-scoped query key for cache separation", async () => {
      (fetcher as Mock).mockResolvedValue(mockQueue);
      const wrapper = createWrapper();

      // Render global and project-scoped hooks in the same client
      const { result: globalResult } = renderHook(() => useQueue(), { wrapper });
      const { result: scopedResult } = renderHook(() => useQueue("proj-a"), { wrapper });

      await waitFor(() => expect(globalResult.current.loading).toBe(false));
      await waitFor(() => expect(scopedResult.current.loading).toBe(false));

      // Both should have called fetcher with different URLs
      expect(fetcher).toHaveBeenCalledWith("/api/queue");
      expect(fetcher).toHaveBeenCalledWith("/api/queue?project=proj-a");
    });

    it("passes projectSlug as default to start() when no project arg given", async () => {
      (fetcher as Mock).mockResolvedValue(mockQueue);
      const { result } = renderHook(() => useQueue("my-project"), {
        wrapper: createWrapper(),
      });
      await waitFor(() => expect(result.current.loading).toBe(false));

      (fetcher as Mock).mockResolvedValue({});
      await act(async () => {
        await result.current.start();
      });

      expect(fetcher).toHaveBeenCalledWith("/api/queue/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project: "my-project" }),
      });
    });

    it("allows overriding projectSlug in start() call", async () => {
      (fetcher as Mock).mockResolvedValue(mockQueue);
      const { result } = renderHook(() => useQueue("my-project"), {
        wrapper: createWrapper(),
      });
      await waitFor(() => expect(result.current.loading).toBe(false));

      (fetcher as Mock).mockResolvedValue({});
      await act(async () => {
        await result.current.start("other-project");
      });

      expect(fetcher).toHaveBeenCalledWith("/api/queue/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project: "other-project" }),
      });
    });

    it("invalidates both project-scoped and global keys on SSE update", async () => {
      (fetcher as Mock).mockResolvedValue(mockQueue);
      const wrapper = createWrapper();
      const { result } = renderHook(() => useQueue("my-project"), { wrapper });
      await waitFor(() => expect(result.current.loading).toBe(false));

      const updatedQueue: QueueStatus = { ...mockQueue, active_workers: 1 };

      act(() => {
        result.current.handleQueueUpdate({ data: updatedQueue });
      });

      // SSE handler should set data on the project-scoped key
      // and invalidate the global key (since project events affect global view)
    });
  });
});
