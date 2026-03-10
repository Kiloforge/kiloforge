import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { createWrapper } from "../test/helpers";
import { useAgents } from "./useAgents";

vi.mock("../api/fetcher", () => ({
  fetcher: vi.fn(),
}));

import { fetcher } from "../api/fetcher";

describe("useAgents", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches active agents via paginated endpoint", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "a1", status: "running" }],
      total_count: 1,
    });

    const { result } = renderHook(() => useAgents(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.agents).toEqual([{ id: "a1", status: "running" }]);
    expect(result.current.totalCount).toBe(1);
    expect(fetcher).toHaveBeenCalledWith("/api/agents?limit=50");
  });

  it("fetches all agents when active=false", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "a2", status: "completed" }],
      total_count: 1,
    });

    const { result } = renderHook(() => useAgents(false), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.agents).toEqual([{ id: "a2", status: "completed" }]);
    expect(fetcher).toHaveBeenCalledWith("/api/agents?active=false&limit=50");
  });

  it("exposes pagination state", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [{ id: "a1" }, { id: "a2" }],
      next_cursor: "cur1",
      total_count: 10,
    });

    const { result } = renderHook(() => useAgents(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.shownCount).toBe(2);
    expect(result.current.remainingCount).toBe(8);
    expect(result.current.hasNextPage).toBe(true);
  });

  it("provides SSE handler functions", async () => {
    (fetcher as Mock).mockResolvedValue({
      items: [],
      total_count: 0,
    });

    const { result } = renderHook(() => useAgents(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(typeof result.current.handleAgentUpdate).toBe("function");
    expect(typeof result.current.handleAgentRemoved).toBe("function");
  });
});
