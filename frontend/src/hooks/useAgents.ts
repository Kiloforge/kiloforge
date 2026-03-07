import { useState, useEffect, useCallback } from "react";
import type { Agent, SSEEventData } from "../types/api";

interface UseAgentsResult {
  agents: Agent[];
  loading: boolean;
  handleAgentUpdate: (raw: unknown) => void;
  handleAgentRemoved: (raw: unknown) => void;
}

export function useAgents(): UseAgentsResult {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/-/api/agents")
      .then((r) => r.json())
      .then((data: Agent[]) => {
        setAgents(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleAgentUpdate = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const agent = event.data as Agent;
    setAgents((prev) => {
      const idx = prev.findIndex((a) => a.id === agent.id);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = { ...next[idx], ...agent };
        return next;
      }
      return [...prev, agent];
    });
  }, []);

  const handleAgentRemoved = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const { id } = event.data as { id: string };
    setAgents((prev) => prev.filter((a) => a.id !== id));
  }, []);

  return { agents, loading, handleAgentUpdate, handleAgentRemoved };
}
