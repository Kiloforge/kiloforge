import { useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { SwarmStatus, SwarmSettings, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseSwarmResult {
  swarm: SwarmStatus | null;
  loading: boolean;
  starting: boolean;
  stopping: boolean;
  updatingSettings: boolean;
  start: (project?: string) => Promise<void>;
  stop: () => Promise<void>;
  updateSettings: (settings: SwarmSettings) => Promise<void>;
  handleSwarmUpdate: (raw: unknown) => void;
}

export function useSwarm(projectSlug?: string): UseSwarmResult {
  const queryClient = useQueryClient();

  const { data: swarm = null, isLoading } = useQuery({
    queryKey: queryKeys.swarm(projectSlug),
    queryFn: () =>
      fetcher<SwarmStatus>(
        projectSlug ? `/api/swarm?project=${projectSlug}` : "/api/swarm",
      ),
  });

  const startMutation = useMutation({
    mutationFn: (project?: string) =>
      fetcher("/api/swarm/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(project ? { project } : {}),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.swarm(projectSlug) });
      if (projectSlug) {
        queryClient.invalidateQueries({ queryKey: queryKeys.swarm() });
      }
    },
  });

  const stopMutation = useMutation({
    mutationFn: () =>
      fetcher("/api/swarm/stop", { method: "POST" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.swarm(projectSlug) });
      if (projectSlug) {
        queryClient.invalidateQueries({ queryKey: queryKeys.swarm() });
      }
    },
  });

  const settingsMutation = useMutation({
    mutationFn: (settings: SwarmSettings) =>
      fetcher("/api/swarm/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(settings),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.swarm(projectSlug) });
      if (projectSlug) {
        queryClient.invalidateQueries({ queryKey: queryKeys.swarm() });
      }
    },
  });

  const start = async (project?: string): Promise<void> => {
    await startMutation.mutateAsync(project ?? projectSlug);
  };

  const stop = async (): Promise<void> => {
    await stopMutation.mutateAsync();
  };

  const updateSettings = async (settings: SwarmSettings): Promise<void> => {
    await settingsMutation.mutateAsync(settings);
  };

  const handleSwarmUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as SwarmStatus;
      if (data && typeof data.running === "boolean") {
        queryClient.setQueryData<SwarmStatus>(queryKeys.swarm(projectSlug), data);
        if (projectSlug) {
          queryClient.invalidateQueries({ queryKey: queryKeys.swarm() });
        }
      } else {
        queryClient.invalidateQueries({ queryKey: queryKeys.swarm(projectSlug) });
        if (projectSlug) {
          queryClient.invalidateQueries({ queryKey: queryKeys.swarm() });
        }
      }
    },
    [queryClient, projectSlug],
  );

  return {
    swarm,
    loading: isLoading,
    starting: startMutation.isPending,
    stopping: stopMutation.isPending,
    updatingSettings: settingsMutation.isPending,
    start,
    stop,
    updateSettings,
    handleSwarmUpdate,
  };
}
