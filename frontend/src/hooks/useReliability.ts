import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type {
  ReliabilityEvent,
  ReliabilitySummary,
} from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { usePaginatedList } from "./usePaginatedList";
import type { UsePaginatedListResult } from "./usePaginatedList";

interface UseReliabilityEventsResult extends UsePaginatedListResult<ReliabilityEvent> {
  events: ReliabilityEvent[];
  handleReliabilityEvent: (raw: unknown) => void;
}

export function useReliabilityEvents(
  filters?: Record<string, string>,
): UseReliabilityEventsResult {
  const queryClient = useQueryClient();
  const qk = queryKeys.reliabilityEventsPaginated(filters);

  const paginated = usePaginatedList<ReliabilityEvent>({
    queryKey: qk,
    url: "/api/reliability/events",
    params: filters,
  });

  const handleReliabilityEvent = useCallback(
    () => {
      queryClient.invalidateQueries({ queryKey: ["reliability"] });
    },
    [queryClient],
  );

  return {
    ...paginated,
    events: paginated.items,
    handleReliabilityEvent,
  };
}

interface UseReliabilitySummaryResult {
  summary: ReliabilitySummary | null;
  loading: boolean;
}

export function useReliabilitySummary(
  since?: string,
  bucket: string = "hour",
): UseReliabilitySummaryResult {
  const params = new URLSearchParams();
  if (since) params.set("since", since);
  params.set("bucket", bucket);

  const { data: summary = null, isLoading } = useQuery({
    queryKey: queryKeys.reliabilitySummary(since, bucket),
    queryFn: () =>
      fetcher<ReliabilitySummary>(
        `/api/reliability/summary?${params.toString()}`,
      ),
  });

  return { summary, loading: isLoading };
}
