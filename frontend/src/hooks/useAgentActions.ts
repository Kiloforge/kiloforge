import { useMutation, useQueryClient } from "@tanstack/react-query";
import { fetcher } from "../api/fetcher";
import { queryKeys } from "../api/queryKeys";
import type { Agent } from "../types/api";

export function useAgentActions() {
  const qc = useQueryClient();

  const invalidateAgents = (id: string) => {
    qc.invalidateQueries({ queryKey: queryKeys.agents() });
    qc.invalidateQueries({ queryKey: queryKeys.agents(false) });
    qc.invalidateQueries({ queryKey: queryKeys.agent(id) });
  };

  const stop = useMutation({
    mutationFn: (id: string) =>
      fetcher<Agent>(`/api/agents/${encodeURIComponent(id)}/stop`, { method: "POST" }),
    onSuccess: (_data, id) => invalidateAgents(id),
  });

  const resume = useMutation({
    mutationFn: (id: string) =>
      fetcher<Agent>(`/api/agents/${encodeURIComponent(id)}/resume`, { method: "POST" }),
    onSuccess: (_data, id) => invalidateAgents(id),
  });

  const del = useMutation({
    mutationFn: (id: string) =>
      fetcher<void>(`/api/agents/${encodeURIComponent(id)}`, { method: "DELETE" }),
    onSuccess: (_data, id) => invalidateAgents(id),
  });

  return { stop, resume, del };
}

/** Stop is available for running or waiting agents */
export function canStop(agent: Agent): boolean {
  return agent.status === "running" || agent.status === "waiting";
}

/** Resume is available for stopped/completed/failed interactive agents */
export function canResume(agent: Agent): boolean {
  return (
    (agent.status === "stopped" || agent.status === "completed" || agent.status === "failed") &&
    agent.role === "interactive"
  );
}

/** Delete is available when the agent is not running or waiting */
export function canDelete(agent: Agent): boolean {
  return agent.status !== "running" && agent.status !== "waiting";
}
