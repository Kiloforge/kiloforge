import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useCallback } from "react";
import type { TraceSummary, PaginatedResponse, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { usePaginatedList, type UsePaginatedListResult } from "./usePaginatedList";

interface UseTracesResult extends UsePaginatedListResult<TraceSummary> {
  traces: TraceSummary[];
  loading: boolean;
  handleTraceUpdate: (raw: unknown) => void;
}

type InfiniteTraces = InfiniteData<PaginatedResponse<TraceSummary>>;

export function useTraces(): UseTracesResult {
  const queryClient = useQueryClient();
  const qk = queryKeys.tracesPaginated;

  const paginated = usePaginatedList<TraceSummary>({
    queryKey: qk,
    url: "/api/traces",
  });

  const handleTraceUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as TraceSummary;
      if (!data?.trace_id) return;
      queryClient.setQueryData<InfiniteTraces>(qk, (prev) => {
        if (!prev) return prev;
        let found = false;
        const pages = prev.pages.map((page) => {
          const idx = page.items.findIndex((t) => t.trace_id === data.trace_id);
          if (idx >= 0) {
            found = true;
            const items = [...page.items];
            items[idx] = { ...items[idx], ...data };
            return { ...page, items };
          }
          return page;
        });
        if (found) return { ...prev, pages };
        // New trace — invalidate to refetch from server
        queryClient.invalidateQueries({ queryKey: qk });
        return prev;
      });
    },
    [queryClient, qk],
  );

  return {
    ...paginated,
    traces: paginated.items,
    loading: paginated.isLoading,
    handleTraceUpdate,
  };
}
