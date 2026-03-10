import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { TraceSummary, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { usePaginatedList, type UsePaginatedListResult } from "./usePaginatedList";

interface UseTracesResult extends UsePaginatedListResult<TraceSummary> {
  traces: TraceSummary[];
  loading: boolean;
  handleTraceUpdate: (raw: unknown) => void;
}

export function useTraces(): UseTracesResult {
  const queryClient = useQueryClient();
  const qk = queryKeys.tracesPaginated;

  const paginated = usePaginatedList<TraceSummary>({
    queryKey: qk,
    url: "/api/traces",
  });

  const handleTraceUpdate = useCallback(
    (raw: unknown) => {
      const _event = raw as SSEEventData;
      queryClient.invalidateQueries({ queryKey: qk });
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
