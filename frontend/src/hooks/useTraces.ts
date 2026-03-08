import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { TraceSummary, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseTracesResult {
  traces: TraceSummary[];
  loading: boolean;
  handleTraceUpdate: (raw: unknown) => void;
}

export function useTraces(): UseTracesResult {
  const queryClient = useQueryClient();

  const { data: traces = [], isLoading } = useQuery({
    queryKey: queryKeys.traces,
    queryFn: () => fetcher<TraceSummary[]>("/api/traces").then((d) => d ?? []),
  });

  const handleTraceUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as TraceSummary;
      if (!data?.trace_id) return;
      queryClient.setQueryData<TraceSummary[]>(queryKeys.traces, (prev = []) => {
        const idx = prev.findIndex((t) => t.trace_id === data.trace_id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = { ...next[idx], ...data };
          return next;
        }
        return [data, ...prev];
      });
    },
    [queryClient],
  );

  return { traces, loading: isLoading, handleTraceUpdate };
}
