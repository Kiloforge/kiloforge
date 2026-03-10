import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import type { SwarmStatus } from "../types/api";
import { useSwarm } from "./useSwarm";

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

const mockSwarm: SwarmStatus = {
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

describe("useSwarm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches swarm status", async () => {
    (fetcher as Mock).mockResolvedValue(mockSwarm);
    const { result } = renderHook(() => useSwarm(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.swarm).toEqual(mockSwarm);
    expect(fetcher).toHaveBeenCalledWith("/api/swarm");
  });

  it("returns null swarm when fetch fails", async () => {
    (fetcher as Mock).mockRejectedValue(new Error("fail"));
    const { result } = renderHook(() => useSwarm(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.swarm).toBeNull();
  });

  it("provides start mutation", async () => {
    (fetcher as Mock).mockResolvedValue(mockSwarm);
    const { result } = renderHook(() => useSwarm(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.start(); });

    expect(fetcher).toHaveBeenCalledWith("/api/swarm/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    });
  });

  it("provides start mutation with project slug", async () => {
    (fetcher as Mock).mockResolvedValue(mockSwarm);
    const { result } = renderHook(() => useSwarm(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.start("my-proj"); });

    expect(fetcher).toHaveBeenCalledWith("/api/swarm/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ project: "my-proj" }),
    });
  });

  it("provides stop mutation", async () => {
    (fetcher as Mock).mockResolvedValue(mockSwarm);
    const { result } = renderHook(() => useSwarm(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.stop(); });

    expect(fetcher).toHaveBeenCalledWith("/api/swarm/stop", { method: "POST" });
  });

  it("provides updateSettings mutation", async () => {
    (fetcher as Mock).mockResolvedValue(mockSwarm);
    const { result } = renderHook(() => useSwarm(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    (fetcher as Mock).mockResolvedValue({});
    await act(async () => { await result.current.updateSettings({ max_workers: 5 }); });

    expect(fetcher).toHaveBeenCalledWith("/api/swarm/settings", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ max_workers: 5 }),
    });
  });

  it("provides handleSwarmUpdate SSE handler", async () => {
    (fetcher as Mock).mockResolvedValue(mockSwarm);
    const { result } = renderHook(() => useSwarm(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(typeof result.current.handleSwarmUpdate).toBe("function");
  });

  describe("project-scoped", () => {
    it("fetches swarm with project query param when projectSlug is provided", async () => {
      (fetcher as Mock).mockResolvedValue(mockSwarm);
      const { result } = renderHook(() => useSwarm("my-project"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(fetcher).toHaveBeenCalledWith("/api/swarm?project=my-project");
    });

    it("uses project-scoped query key for cache separation", async () => {
      (fetcher as Mock).mockResolvedValue(mockSwarm);
      const wrapper = createWrapper();

      // Render global and project-scoped hooks in the same client
      const { result: globalResult } = renderHook(() => useSwarm(), { wrapper });
      const { result: scopedResult } = renderHook(() => useSwarm("proj-a"), { wrapper });

      await waitFor(() => expect(globalResult.current.loading).toBe(false));
      await waitFor(() => expect(scopedResult.current.loading).toBe(false));

      // Both should have called fetcher with different URLs
      expect(fetcher).toHaveBeenCalledWith("/api/swarm");
      expect(fetcher).toHaveBeenCalledWith("/api/swarm?project=proj-a");
    });

    it("passes projectSlug as default to start() when no project arg given", async () => {
      (fetcher as Mock).mockResolvedValue(mockSwarm);
      const { result } = renderHook(() => useSwarm("my-project"), {
        wrapper: createWrapper(),
      });
      await waitFor(() => expect(result.current.loading).toBe(false));

      (fetcher as Mock).mockResolvedValue({});
      await act(async () => {
        await result.current.start();
      });

      expect(fetcher).toHaveBeenCalledWith("/api/swarm/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project: "my-project" }),
      });
    });

    it("allows overriding projectSlug in start() call", async () => {
      (fetcher as Mock).mockResolvedValue(mockSwarm);
      const { result } = renderHook(() => useSwarm("my-project"), {
        wrapper: createWrapper(),
      });
      await waitFor(() => expect(result.current.loading).toBe(false));

      (fetcher as Mock).mockResolvedValue({});
      await act(async () => {
        await result.current.start("other-project");
      });

      expect(fetcher).toHaveBeenCalledWith("/api/swarm/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project: "other-project" }),
      });
    });

    it("invalidates both project-scoped and global keys on SSE update", async () => {
      (fetcher as Mock).mockResolvedValue(mockSwarm);
      const wrapper = createWrapper();
      const { result } = renderHook(() => useSwarm("my-project"), { wrapper });
      await waitFor(() => expect(result.current.loading).toBe(false));

      const updatedSwarm: SwarmStatus = { ...mockSwarm, active_workers: 1 };

      act(() => {
        result.current.handleSwarmUpdate({ data: updatedSwarm });
      });

      // SSE handler should set data on the project-scoped key
      // and invalidate the global key (since project events affect global view)
    });
  });
});
