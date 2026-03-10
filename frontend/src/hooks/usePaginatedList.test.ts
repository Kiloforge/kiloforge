import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { createWrapper } from "../test/helpers";
import { usePaginatedList } from "./usePaginatedList";

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn(),
}));

import { fetcher } from "../api/fetcher";

describe("usePaginatedList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches first page and exposes items", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "a" }, { id: "b" }],
      next_cursor: "cur1",
      total_count: 5,
    });

    const { result } = renderHook(
      () => usePaginatedList<{ id: string }>({ queryKey: ["test"], url: "/api/test" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.items).toEqual([{ id: "a" }, { id: "b" }]);
    expect(result.current.totalCount).toBe(5);
    expect(result.current.hasNextPage).toBe(true);
    expect(fetcher).toHaveBeenCalledWith("/api/test?limit=50");
  });

  it("returns empty items when response has no items", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [],
      total_count: 0,
    });

    const { result } = renderHook(
      () => usePaginatedList<{ id: string }>({ queryKey: ["empty"], url: "/api/empty" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.items).toEqual([]);
    expect(result.current.totalCount).toBe(0);
    expect(result.current.hasNextPage).toBe(false);
  });

  it("fetches next page with cursor", async () => {
    (fetcher as Mock)
      .mockResolvedValueOnce({
        items: [{ id: "a" }],
        next_cursor: "cur1",
        total_count: 3,
      })
      .mockResolvedValueOnce({
        items: [{ id: "b" }, { id: "c" }],
        total_count: 3,
      });

    const { result } = renderHook(
      () => usePaginatedList<{ id: string }>({ queryKey: ["pages"], url: "/api/pages" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.items).toEqual([{ id: "a" }]);

    await act(async () => {
      await result.current.fetchNextPage();
    });

    await waitFor(() => expect(result.current.items).toHaveLength(3));
    expect(result.current.items).toEqual([{ id: "a" }, { id: "b" }, { id: "c" }]);
    expect(result.current.hasNextPage).toBe(false);
    expect(fetcher).toHaveBeenCalledWith("/api/pages?limit=50&cursor=cur1");
  });

  it("respects custom limit", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "a" }],
      total_count: 1,
    });

    renderHook(
      () => usePaginatedList<{ id: string }>({ queryKey: ["lim"], url: "/api/lim", limit: 10 }),
      { wrapper: createWrapper() },
    );

    await waitFor(() =>
      expect(fetcher).toHaveBeenCalledWith("/api/lim?limit=10"),
    );
  });

  it("appends extra params to URL", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [],
      total_count: 0,
    });

    renderHook(
      () =>
        usePaginatedList<{ id: string }>({
          queryKey: ["params"],
          url: "/api/params",
          params: { active: "true", status: "running" },
        }),
      { wrapper: createWrapper() },
    );

    await waitFor(() =>
      expect(fetcher).toHaveBeenCalledWith(
        "/api/params?active=true&status=running&limit=50",
      ),
    );
  });

  it("computes shownCount from flattened pages", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "a" }, { id: "b" }],
      next_cursor: "x",
      total_count: 10,
    });

    const { result } = renderHook(
      () => usePaginatedList<{ id: string }>({ queryKey: ["shown"], url: "/api/shown" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.shownCount).toBe(2);
    expect(result.current.remainingCount).toBe(8);
  });

  it("can be disabled via enabled option", async () => {
    renderHook(
      () => usePaginatedList<{ id: string }>({ queryKey: ["dis"], url: "/api/dis", enabled: false }),
      { wrapper: createWrapper() },
    );

    // Should not fetch when disabled
    expect(fetcher).not.toHaveBeenCalled();
  });
});
