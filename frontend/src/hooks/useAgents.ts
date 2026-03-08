import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { Agent, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseAgentsResult {
  agents: Agent[];
  loading: boolean;
  handleAgentUpdate: (raw: unknown) => void;
  handleAgentRemoved: (raw: unknown) => void;
}

export function useAgents(): UseAgentsResult {
  const queryClient = useQueryClient();

  const { data: agents = [], isLoading } = useQuery({
    queryKey: queryKeys.agents,
    queryFn: () => fetcher<Agent[]>("/api/agents").then((d) => d || []),
  });

  const handleAgentUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const agent = event.data as Agent;
      queryClient.setQueryData<Agent[]>(queryKeys.agents, (prev = []) => {
        const idx = prev.findIndex((a) => a.id === agent.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = { ...next[idx], ...agent };
          return next;
        }
        return [...prev, agent];
      });
    },
    [queryClient],
  );

  const handleAgentRemoved = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const { id } = event.data as { id: string };
      queryClient.setQueryData<Agent[]>(queryKeys.agents, (prev = []) =>
        prev.filter((a) => a.id !== id),
      );
    },
    [queryClient],
  );

  return { agents, loading: isLoading, handleAgentUpdate, handleAgentRemoved };
}
