import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useTour } from "./useTour";

// Mock fetcher
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
const mockFetcher = vi.mocked(fetcher);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useTour", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("sends action 'accept' body when starting tour", async () => {
    mockFetcher.mockResolvedValueOnce({ status: "pending", current_step: 0 });
    mockFetcher.mockResolvedValueOnce({ status: "active", current_step: 0 });

    const { result } = renderHook(() => useTour(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isPending).toBe(true));

    act(() => result.current.startTour());

    await waitFor(() => {
      expect(mockFetcher).toHaveBeenCalledWith("/api/tour", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "accept" }),
      });
    });
  });

  it("sends action 'advance' with step number", async () => {
    mockFetcher.mockResolvedValueOnce({ status: "active", current_step: 0 });
    mockFetcher.mockResolvedValueOnce({ status: "active", current_step: 3 });

    const { result } = renderHook(() => useTour(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isActive).toBe(true));

    act(() => result.current.advanceStep(3));

    await waitFor(() => {
      expect(mockFetcher).toHaveBeenCalledWith("/api/tour", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "advance", step: 3 }),
      });
    });
  });

  it("sends action 'dismiss' when dismissing", async () => {
    mockFetcher.mockResolvedValueOnce({ status: "active", current_step: 0 });
    mockFetcher.mockResolvedValueOnce({ status: "dismissed", current_step: 0 });

    const { result } = renderHook(() => useTour(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isActive).toBe(true));

    act(() => result.current.dismissTour());

    await waitFor(() => {
      expect(mockFetcher).toHaveBeenCalledWith("/api/tour", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "dismiss" }),
      });
    });
  });

  it("sends action 'complete' when completing", async () => {
    mockFetcher.mockResolvedValueOnce({ status: "active", current_step: 6 });
    mockFetcher.mockResolvedValueOnce({ status: "completed", current_step: 6 });

    const { result } = renderHook(() => useTour(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isActive).toBe(true));

    act(() => result.current.completeTour());

    await waitFor(() => {
      expect(mockFetcher).toHaveBeenCalledWith("/api/tour", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "complete" }),
      });
    });
  });
});
