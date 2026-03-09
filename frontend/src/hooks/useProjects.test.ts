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
import { useProjects } from "./useProjects";

const mockFetcher = vi.mocked(fetcher);

describe("useProjects", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches project list on mount", async () => {
    const projects = [{ slug: "proj-1", repo_name: "proj-1", active: true }];
    mockFetcher.mockResolvedValueOnce(projects);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.projects).toEqual(projects);
    expect(mockFetcher).toHaveBeenCalledWith("/api/projects");
  });

  it("returns empty array when fetch returns null", async () => {
    mockFetcher.mockResolvedValueOnce(null);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.projects).toEqual([]);
  });

  it("addProject sends correct request", async () => {
    mockFetcher.mockResolvedValueOnce([]);
    mockFetcher.mockResolvedValueOnce({ slug: "new-proj", repo_name: "new-proj", active: true });

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    let success: boolean = false;
    await act(async () => {
      success = await result.current.addProject({ remote_url: "https://github.com/test/repo" });
    });

    expect(success).toBe(true);
    expect(mockFetcher).toHaveBeenCalledWith("/api/projects", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ remote_url: "https://github.com/test/repo" }),
    });
  });

  it("addProject sets error on FetchError", async () => {
    mockFetcher.mockResolvedValueOnce([]);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(new FetchError(409, { error: "Already exists" }));

    let success: boolean = true;
    await act(async () => {
      success = await result.current.addProject({ remote_url: "https://github.com/test/repo" });
    });

    expect(success).toBe(false);
    expect(result.current.error).toBe("Already exists");
  });

  it("addProject sets generic error on network failure", async () => {
    mockFetcher.mockResolvedValueOnce([]);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetcher.mockRejectedValueOnce(new Error("Network error"));

    await act(async () => {
      await result.current.addProject({ remote_url: "https://github.com/test/repo" });
    });

    expect(result.current.error).toBe("Network error");
  });

  it("removeProject sends correct request", async () => {
    mockFetcher.mockResolvedValueOnce([{ slug: "proj-1", repo_name: "proj-1", active: true }]);
    mockFetcher.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    let success: boolean = false;
    await act(async () => {
      success = await result.current.removeProject("proj-1", true);
    });

    expect(success).toBe(true);
    expect(mockFetcher).toHaveBeenCalledWith("/api/projects/proj-1?cleanup=true", { method: "DELETE" });
  });

  it("removeProject without cleanup omits query param", async () => {
    mockFetcher.mockResolvedValueOnce([]);
    mockFetcher.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.removeProject("proj-1", false);
    });

    expect(mockFetcher).toHaveBeenCalledWith("/api/projects/proj-1", { method: "DELETE" });
  });

  it("handleProjectUpdate adds new project to cache", async () => {
    mockFetcher.mockResolvedValueOnce([{ slug: "proj-1", repo_name: "proj-1", active: true }]);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => {
      result.current.handleProjectUpdate({ type: "project.updated", data: { slug: "proj-2", repo_name: "proj-2", active: true } });
    });

    await waitFor(() => expect(result.current.projects).toHaveLength(2));
    expect(result.current.projects[1].slug).toBe("proj-2");
  });

  it("handleProjectUpdate updates existing project in cache", async () => {
    mockFetcher.mockResolvedValueOnce([{ slug: "proj-1", repo_name: "proj-1", active: true }]);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => {
      result.current.handleProjectUpdate({ type: "project.updated", data: { slug: "proj-1", repo_name: "renamed", active: true } });
    });

    await waitFor(() => expect(result.current.projects[0].repo_name).toBe("renamed"));
  });

  it("handleProjectRemoved removes project from cache", async () => {
    mockFetcher.mockResolvedValueOnce([
      { slug: "proj-1", repo_name: "proj-1", active: true },
      { slug: "proj-2", repo_name: "proj-2", active: true },
    ]);

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => {
      result.current.handleProjectRemoved({ type: "project.removed", data: { slug: "proj-1" } });
    });

    await waitFor(() => expect(result.current.projects).toHaveLength(1));
    expect(result.current.projects[0].slug).toBe("proj-2");
  });

  it("clearError resets error state", async () => {
    mockFetcher.mockResolvedValueOnce([]);
    mockFetcher.mockRejectedValueOnce(new FetchError(500, { error: "Server error" }));

    const { result } = renderHook(() => useProjects(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.addProject({ remote_url: "test" });
    });
    expect(result.current.error).toBe("Server error");

    act(() => result.current.clearError());
    expect(result.current.error).toBeNull();
  });
});
