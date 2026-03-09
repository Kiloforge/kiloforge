import { useMemo } from "react";
import type { Agent, Track } from "../types/api";
import { AgentCard } from "./AgentCard";
import styles from "./AgentGrid.module.css";

interface Props {
  agents: Agent[];
  tracks?: Track[];
  onViewLog: (agentId: string) => void;
  onAttach?: (agentId: string) => void;
}

const statusOrder: Record<string, number> = {
  running: 0,
  waiting: 1,
  halted: 2,
  suspended: 3,
  suspending: 3,
  stopped: 4,
  completed: 5,
  failed: 6,
};

function sortAgents(agents: Agent[]): Agent[] {
  return [...agents].sort((a, b) => {
    const orderA = statusOrder[a.status] ?? 99;
    const orderB = statusOrder[b.status] ?? 99;
    if (orderA !== orderB) return orderA - orderB;
    return (b.updated_at || "").localeCompare(a.updated_at || "");
  });
}

export function AgentGrid({ agents, tracks, onViewLog, onAttach }: Props) {
  const trackProjectMap = useMemo(() => {
    if (!tracks) return new Map<string, string>();
    const map = new Map<string, string>();
    for (const t of tracks) {
      if (t.project) map.set(t.id, t.project);
    }
    return map;
  }, [tracks]);

  if (agents.length === 0) {
    return <p className={styles.empty}>No agents running</p>;
  }

  return (
    <div className={styles.grid}>
      {sortAgents(agents).map((agent) => (
        <AgentCard
          key={agent.id}
          agent={agent}
          onViewLog={onViewLog}
          onAttach={onAttach}
          projectSlug={agent.ref ? trackProjectMap.get(agent.ref) : undefined}
        />
      ))}
    </div>
  );
}
