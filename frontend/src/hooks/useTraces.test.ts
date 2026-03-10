import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { createWrapper } from "../test/helpers";
import { useTraces } from "./useTraces";

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn(),
}));

import { fetcher } from "../api/fetcher";

describe("useTraces", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches traces via paginated endpoint", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ trace_id: "t1", root_name: "span/1", span_count: 2, start_time: "2026-01-01T00:00:00Z", end_time: "2026-01-01T00:01:00Z" }],
      total_count: 1,
    });

    const { result } = renderHook(() => useTraces(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.traces).toHaveLength(1);
    expect(result.current.traces[0].trace_id).toBe("t1");
    expect(result.current.totalCount).toBe(1);
    expect(fetcher).toHaveBeenCalledWith("/api/traces?limit=50");
  });

  it("exposes pagination state when more pages exist", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ trace_id: "t1" }],
      next_cursor: "c1",
      total_count: 20,
    });

    const { result } = renderHook(() => useTraces(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.remainingCount).toBe(19);
    expect(result.current.hasNextPage).toBe(true);
  });

  it("provides SSE handler function", async () => {
    (fetcher as Mock).mockResolvedValue({ items: [], total_count: 0 });

    const { result } = renderHook(() => useTraces(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(typeof result.current.handleTraceUpdate).toBe("function");
  });
});
