import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import type { SyncStatus, PushRequest, PullResult, PushResult } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher, FetchError } from "../api/fetcher";

interface UseOriginSyncResult {
  syncStatus: SyncStatus | null;
  loading: boolean;
  pushing: boolean;
  pulling: boolean;
  error: string | null;
  push: (req: PushRequest) => Promise<PushResult | null>;
  pull: (remoteBranch?: string) => Promise<PullResult | null>;
  refresh: () => void;
  clearError: () => void;
}

export function useOriginSync(slug?: string): UseOriginSyncResult {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const key = queryKeys.syncStatus(slug ?? "");

  const { data: syncStatus = null, isLoading, refetch } = useQuery({
    queryKey: key,
    queryFn: () =>
      fetcher<SyncStatus>(`/api/projects/${encodeURIComponent(slug!)}/sync-status`),
    enabled: !!slug,
  });

  const pushMutation = useMutation({
    mutationFn: (req: PushRequest) =>
      fetcher<PushResult>(`/api/projects/${encodeURIComponent(slug!)}/push`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: key });
    },
    onError: (err) => {
      if (err instanceof FetchError) {
        const body = err.body as { error?: string };
        setError(body?.error || `Error ${err.status}`);
      } else {
        setError("Network error");
      }
    },
  });

  const pullMutation = useMutation({
    mutationFn: (remoteBranch?: string) =>
      fetcher<PullResult>(`/api/projects/${encodeURIComponent(slug!)}/pull`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ remote_branch: remoteBranch }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: key });
    },
    onError: (err) => {
      if (err instanceof FetchError) {
        const body = err.body as { error?: string };
        setError(body?.error || `Error ${err.status}`);
      } else {
        setError("Network error");
      }
    },
  });

  const push = async (req: PushRequest): Promise<PushResult | null> => {
    setError(null);
    try {
      return await pushMutation.mutateAsync(req);
    } catch {
      return null;
    }
  };

  const pull = async (remoteBranch?: string): Promise<PullResult | null> => {
    setError(null);
    try {
      return await pullMutation.mutateAsync(remoteBranch);
    } catch {
      return null;
    }
  };

  const clearError = useCallback(() => setError(null), []);

  return {
    syncStatus,
    loading: isLoading,
    pushing: pushMutation.isPending,
    pulling: pullMutation.isPending,
    error,
    push,
    pull,
    refresh: () => { refetch(); },
    clearError,
  };
}
