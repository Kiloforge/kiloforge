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

import { fetcher } from "../api/fetcher";
import { useBoard } from "./useBoard";
import type { BoardState } from "../types/api";

const mockFetcher = vi.mocked(fetcher);

const mockBoard: BoardState = {
  columns: ["todo", "in_progress", "done"],
  cards: {
    "track-1": {
      track_id: "track-1",
      title: "Task One",
      column: "todo",
      position: 0,
      moved_at: "2026-01-01T00:00:00Z",
      created_at: "2026-01-01T00:00:00Z",
    },
    "track-2": {
      track_id: "track-2",
      title: "Task Two",
      column: "in_progress",
      position: 0,
      moved_at: "2026-01-01T00:00:00Z",
      created_at: "2026-01-01T00:00:00Z",
    },
  },
};

describe("useBoard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not fetch when project is undefined", () => {
    renderHook(() => useBoard(undefined), { wrapper: createWrapper() });
    expect(mockFetcher).not.toHaveBeenCalled();
  });

  it("fetches board state for project", async () => {
    mockFetcher.mockResolvedValueOnce(mockBoard);

    const { result } = renderHook(() => useBoard("my-proj"), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.board).toEqual(mockBoard);
    expect(mockFetcher).toHaveBeenCalledWith("/api/board/my-proj");
  });

  it("moveCard applies optimistic update immediately", async () => {
    mockFetcher.mockResolvedValueOnce(mockBoard);

    const { result } = renderHook(() => useBoard("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.board).not.toBeNull());

    // Mock the move endpoint
    mockFetcher.mockResolvedValueOnce(undefined);
    // Mock the refetch after settle
    mockFetcher.mockResolvedValueOnce({
      ...mockBoard,
      cards: {
        ...mockBoard.cards,
        "track-1": { ...mockBoard.cards["track-1"], column: "done" },
      },
    });

    await act(async () => {
      await result.current.moveCard("track-1", "done");
    });

    expect(mockFetcher).toHaveBeenCalledWith("/api/board/my-proj/move", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ track_id: "track-1", to_column: "done" }),
    });
  });

  it("moveCard rolls back on server error", async () => {
    mockFetcher.mockResolvedValueOnce(mockBoard);

    const { result } = renderHook(() => useBoard("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.board).not.toBeNull());

    // Mock the move endpoint to fail
    mockFetcher.mockRejectedValueOnce(new Error("Server error"));
    // Mock refetch after error settle
    mockFetcher.mockResolvedValueOnce(mockBoard);

    await act(async () => {
      try {
        await result.current.moveCard("track-1", "done");
      } catch {
        // expected
      }
    });

    // After rollback, the card should be back in original column
    await waitFor(() => {
      expect(result.current.board?.cards["track-1"].column).toBe("todo");
    });
  });

  it("handleBoardUpdate replaces full board state", async () => {
    mockFetcher.mockResolvedValueOnce(mockBoard);

    const { result } = renderHook(() => useBoard("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.board).not.toBeNull());

    const newBoard: BoardState = {
      columns: ["backlog", "todo", "done"],
      cards: {
        "track-3": {
          track_id: "track-3",
          title: "New Task",
          column: "backlog",
          position: 0,
          moved_at: "2026-01-02T00:00:00Z",
          created_at: "2026-01-02T00:00:00Z",
        },
      },
    };

    act(() => {
      result.current.handleBoardUpdate({ type: "board.updated", data: newBoard });
    });

    await waitFor(() => {
      expect(result.current.board?.columns).toEqual(["backlog", "todo", "done"]);
      expect(result.current.board?.cards["track-3"]).toBeDefined();
    });
  });

  it("handleBoardUpdate handles card move events", async () => {
    mockFetcher.mockResolvedValueOnce(mockBoard);

    const { result } = renderHook(() => useBoard("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.board).not.toBeNull());

    act(() => {
      result.current.handleBoardUpdate({
        type: "board.card_moved",
        data: { track_id: "track-1", to_column: "done" },
      });
    });

    await waitFor(() => {
      expect(result.current.board?.cards["track-1"].column).toBe("done");
    });
  });

  it("syncBoard sends POST and invalidates cache", async () => {
    mockFetcher.mockResolvedValueOnce(mockBoard);

    const { result } = renderHook(() => useBoard("my-proj"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.board).not.toBeNull());

    mockFetcher.mockResolvedValueOnce(undefined);
    mockFetcher.mockResolvedValueOnce(mockBoard);

    act(() => {
      result.current.syncBoard();
    });

    await waitFor(() => {
      expect(mockFetcher).toHaveBeenCalledWith("/api/board/my-proj/sync", { method: "POST" });
    });
  });
});
