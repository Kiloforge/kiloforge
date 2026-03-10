import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { Agent, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { usePaginatedList, type UsePaginatedListResult } from "./usePaginatedList";

interface UseAgentsResult extends UsePaginatedListResult<Agent> {
  agents: Agent[];
  loading: boolean;
  handleAgentUpdate: (raw: unknown) => void;
  handleAgentRemoved: (raw: unknown) => void;
}

export function useAgents(active = true): UseAgentsResult {
  const queryClient = useQueryClient();
  const qk = queryKeys.agentsPaginated(active ? undefined : false);

  const paginated = usePaginatedList<Agent>({
    queryKey: qk,
    url: "/api/agents",
    params: active ? undefined : { active: "false" },
  });

  const handleAgentUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const _agent = event.data as Agent;
      // Invalidate to refetch — SSE in-place updates handled in Phase 3
      queryClient.invalidateQueries({ queryKey: qk });
    },
    [queryClient, qk],
  );

  const handleAgentRemoved = useCallback(
    (raw: unknown) => {
      const _event = raw as SSEEventData;
      queryClient.invalidateQueries({ queryKey: qk });
    },
    [queryClient, qk],
  );

  return {
    ...paginated,
    agents: paginated.items,
    loading: paginated.isLoading,
    handleAgentUpdate,
    handleAgentRemoved,
  };
}
