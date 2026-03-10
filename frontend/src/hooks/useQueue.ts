import { useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { QueueStatus, QueueSettings, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseQueueResult {
  queue: QueueStatus | null;
  loading: boolean;
  starting: boolean;
  stopping: boolean;
  updatingSettings: boolean;
  start: (project?: string) => Promise<void>;
  stop: () => Promise<void>;
  updateSettings: (settings: QueueSettings) => Promise<void>;
  handleQueueUpdate: (raw: unknown) => void;
}

export function useQueue(projectSlug?: string): UseQueueResult {
  const queryClient = useQueryClient();

  const { data: queue = null, isLoading } = useQuery({
    queryKey: queryKeys.queue(projectSlug),
    queryFn: () =>
      fetcher<QueueStatus>(
        projectSlug ? `/api/queue?project=${projectSlug}` : "/api/queue",
      ),
  });

  const startMutation = useMutation({
    mutationFn: (project?: string) =>
      fetcher("/api/queue/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(project ? { project } : {}),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.queue(projectSlug) });
      if (projectSlug) {
        queryClient.invalidateQueries({ queryKey: queryKeys.queue() });
      }
    },
  });

  const stopMutation = useMutation({
    mutationFn: () =>
      fetcher("/api/queue/stop", { method: "POST" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.queue(projectSlug) });
      if (projectSlug) {
        queryClient.invalidateQueries({ queryKey: queryKeys.queue() });
      }
    },
  });

  const settingsMutation = useMutation({
    mutationFn: (settings: QueueSettings) =>
      fetcher("/api/queue/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(settings),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.queue(projectSlug) });
      if (projectSlug) {
        queryClient.invalidateQueries({ queryKey: queryKeys.queue() });
      }
    },
  });

  const start = async (project?: string): Promise<void> => {
    await startMutation.mutateAsync(project ?? projectSlug);
  };

  const stop = async (): Promise<void> => {
    await stopMutation.mutateAsync();
  };

  const updateSettings = async (settings: QueueSettings): Promise<void> => {
    await settingsMutation.mutateAsync(settings);
  };

  const handleQueueUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as QueueStatus;
      if (data && typeof data.running === "boolean") {
        queryClient.setQueryData<QueueStatus>(queryKeys.queue(projectSlug), data);
        if (projectSlug) {
          queryClient.invalidateQueries({ queryKey: queryKeys.queue() });
        }
      } else {
        queryClient.invalidateQueries({ queryKey: queryKeys.queue(projectSlug) });
        if (projectSlug) {
          queryClient.invalidateQueries({ queryKey: queryKeys.queue() });
        }
      }
    },
    [queryClient, projectSlug],
  );

  return {
    queue,
    loading: isLoading,
    starting: startMutation.isPending,
    stopping: stopMutation.isPending,
    updatingSettings: settingsMutation.isPending,
    start,
    stop,
    updateSettings,
    handleQueueUpdate,
  };
}
