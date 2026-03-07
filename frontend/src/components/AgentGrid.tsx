import type { Agent } from "../types/api";
import { AgentCard } from "./AgentCard";
import styles from "./AgentGrid.module.css";

interface Props {
  agents: Agent[];
  giteaURL: string;
  onViewLog: (agentId: string) => void;
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

export function AgentGrid({ agents, giteaURL, onViewLog }: Props) {
  if (agents.length === 0) {
    return <p className={styles.empty}>No agents running</p>;
  }

  return (
    <div className={styles.grid}>
      {sortAgents(agents).map((agent) => (
        <AgentCard key={agent.id} agent={agent} giteaURL={giteaURL} onViewLog={onViewLog} />
      ))}
    </div>
  );
}
