import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { createWrapper } from "../test/helpers";

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn(),
  FetchError: class FetchError extends Error {
    status: number;
    body: unknown;
    constructor(status: number, body: unknown) {
      super(`Request failed with status ${status}`);
      this.name = "FetchError";
      this.status = status;
      this.body = body;
    }
  },
}));

import { fetcher, FetchError } from "../api/fetcher";
import { useOriginSync } from "./useOriginSync";

const mockFetcher = vi.mocked(fetcher);

const mockSyncStatus = {
  ahead: 2,
  behind: 1,
  status: "ahead" as const,
  local_branch: "main",
  remote_url: "https://github.com/test/repo",
};

describe("useOriginSync", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not fetch when slug is undefined", () => {
    renderHook(() => useOriginSync(undefined), { wrapper: createWrapper() });
    expect(mockFetcher).not.toHaveBeenCalled();
  });

  it("fetches sync status with correct URL", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.syncStatus).toEqual(mockSyncStatus);
    expect(mockFetcher).toHaveBeenCalledWith("/api/projects/my-proj/sync-status");
  });

  it("push sends correct request and returns result", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    const pushResult = { success: true, local_branch: "main", remote_branch: "main" };
    mockFetcher.mockResolvedValueOnce(pushResult);
    // Mock the refetch after invalidation
    mockFetcher.mockResolvedValueOnce({ ...mockSyncStatus, ahead: 0, status: "synced" });

    let res: unknown = null;
    await act(async () => {
      res = await result.current.push({ remote_branch: "main", force: false });
    });

    expect(res).toEqual(pushResult);
    expect(mockFetcher).toHaveBeenCalledWith("/api/projects/my-proj/push", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ remote_branch: "main", force: false }),
    });
  });

  it("pull sends correct request and returns result", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    const pullResult = { success: true, new_head: "abc123" };
    mockFetcher.mockResolvedValueOnce(pullResult);
    mockFetcher.mockResolvedValueOnce({ ...mockSyncStatus, behind: 0 });

    let res: unknown = null;
    await act(async () => {
      res = await result.current.pull("main");
    });

    expect(res).toEqual(pullResult);
    expect(mockFetcher).toHaveBeenCalledWith("/api/projects/my-proj/pull", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ remote_branch: "main" }),
    });
  });

  it("pull without remote branch sends undefined", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockResolvedValueOnce({ success: true, new_head: "abc" });
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    await act(async () => {
      await result.current.pull();
    });

    expect(mockFetcher).toHaveBeenCalledWith("/api/projects/my-proj/pull", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ remote_branch: undefined }),
    });
  });

  it("push sets error on FetchError", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(new FetchError(403, { error: "Access denied" }));

    let res: unknown = "initial";
    await act(async () => {
      res = await result.current.push({ remote_branch: "main" });
    });

    expect(res).toBeNull();
    expect(result.current.error).toBe("Access denied");
  });

  it("pull sets error on network failure", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(new Error("Network error"));

    await act(async () => {
      await result.current.pull();
    });

    expect(result.current.error).toBe("Network error");
  });

  it("push 409 sets conflict state with push direction", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(
      new FetchError(409, { error: "Branches have diverged", direction: "push" }),
    );

    await act(async () => {
      await result.current.push({ remote_branch: "kf/main" });
    });

    expect(result.current.conflict).toEqual({ active: true, direction: "push" });
    expect(result.current.error).toBeNull();
  });

  it("pull 409 sets conflict state with pull direction", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(
      new FetchError(409, { error: "Branches have diverged", direction: "pull" }),
    );

    await act(async () => {
      await result.current.pull();
    });

    expect(result.current.conflict).toEqual({ active: true, direction: "pull" });
    expect(result.current.error).toBeNull();
  });

  it("clearConflict resets conflict state", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(
      new FetchError(409, { error: "Diverged", direction: "push" }),
    );

    await act(async () => {
      await result.current.push({ remote_branch: "kf/main" });
    });
    expect(result.current.conflict?.active).toBe(true);

    act(() => result.current.clearConflict());
    expect(result.current.conflict).toBeNull();
  });

  it("non-409 push error still sets generic error", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(new FetchError(500, { error: "Server error" }));

    await act(async () => {
      await result.current.push({ remote_branch: "kf/main" });
    });

    expect(result.current.conflict).toBeNull();
    expect(result.current.error).toBe("Server error");
  });

  it("clearError resets error state", async () => {
    mockFetcher.mockResolvedValueOnce(mockSyncStatus);

    const { result } = renderHook(() => useOriginSync("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(new FetchError(500, { error: "Server error" }));

    await act(async () => {
      await result.current.push({ remote_branch: "main" });
    });
    expect(result.current.error).toBe("Server error");

    act(() => result.current.clearError());
    expect(result.current.error).toBeNull();
  });
});
