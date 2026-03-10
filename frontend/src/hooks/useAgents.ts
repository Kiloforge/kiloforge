import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useCallback } from "react";
import type { Agent, PaginatedResponse, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { usePaginatedList, type UsePaginatedListResult } from "./usePaginatedList";

interface UseAgentsResult extends UsePaginatedListResult<Agent> {
  agents: Agent[];
  loading: boolean;
  handleAgentUpdate: (raw: unknown) => void;
  handleAgentRemoved: (raw: unknown) => void;
}

type InfiniteAgents = InfiniteData<PaginatedResponse<Agent>>;

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
      const agent = event.data as Agent;
      queryClient.setQueryData<InfiniteAgents>(qk, (prev) => {
        if (!prev) return prev;
        let found = false;
        const pages = prev.pages.map((page) => {
          const idx = page.items.findIndex((a) => a.id === agent.id);
          if (idx >= 0) {
            found = true;
            const items = [...page.items];
            items[idx] = { ...items[idx], ...agent };
            return { ...page, items };
          }
          return page;
        });
        if (found) return { ...prev, pages };
        // New agent — invalidate to refetch from server
        queryClient.invalidateQueries({ queryKey: qk });
        return prev;
      });
    },
    [queryClient, qk],
  );

  const handleAgentRemoved = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const { id } = event.data as { id: string };
      queryClient.setQueryData<InfiniteAgents>(qk, (prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          pages: prev.pages.map((page) => ({
            ...page,
            items: page.items.filter((a) => a.id !== id),
            total_count: Math.max(0, page.total_count - (page.items.some((a) => a.id === id) ? 1 : 0)),
          })),
        };
      });
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
