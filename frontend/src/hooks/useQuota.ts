import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { QuotaResponse, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseQuotaResult {
  quota: QuotaResponse | null;
  loading: boolean;
  handleQuotaUpdate: (raw: unknown) => void;
}

export function useQuota(): UseQuotaResult {
  const queryClient = useQueryClient();

  const { data: quota = null, isLoading } = useQuery({
    queryKey: queryKeys.quota,
    queryFn: () => fetcher<QuotaResponse>("/api/quota"),
  });

  const handleQuotaUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      queryClient.setQueryData<QuotaResponse>(
        queryKeys.quota,
        event.data as QuotaResponse,
      );
    },
    [queryClient],
  );

  return { quota, loading: isLoading, handleQuotaUpdate };
}
