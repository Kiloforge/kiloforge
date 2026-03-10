import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { createWrapper } from "../test/helpers";
import { useTracks } from "./useTracks";

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn(),
}));

import { fetcher } from "../api/fetcher";

describe("useTracks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches all tracks via paginated endpoint", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "t1", title: "Track 1", status: "pending" }],
      total_count: 1,
    });

    const { result } = renderHook(() => useTracks(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.tracks).toEqual([{ id: "t1", title: "Track 1", status: "pending" }]);
    expect(result.current.totalCount).toBe(1);
    expect(fetcher).toHaveBeenCalledWith("/api/tracks?limit=50");
  });

  it("passes project param when provided", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [],
      total_count: 0,
    });

    renderHook(() => useTracks("my-proj"), { wrapper: createWrapper() });
    await waitFor(() =>
      expect(fetcher).toHaveBeenCalledWith("/api/tracks?project=my-proj&limit=50"),
    );
  });

  it("exposes pagination state", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "t1" }],
      next_cursor: "c1",
      total_count: 5,
    });

    const { result } = renderHook(() => useTracks(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.remainingCount).toBe(4);
    expect(result.current.hasNextPage).toBe(true);
  });

  it("provides SSE handler functions", async () => {
    (fetcher as Mock).mockResolvedValue({ items: [], total_count: 0 });

    const { result } = renderHook(() => useTracks(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(typeof result.current.handleTrackUpdate).toBe("function");
    expect(typeof result.current.handleTrackRemoved).toBe("function");
  });
});
