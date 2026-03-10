import type { Agent } from "../types/api";
import { AgentCard } from "./AgentCard";
import styles from "./AgentGrid.module.css";

interface Props {
  agents: Agent[];
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

export function AgentGrid({ agents, onViewLog, onAttach }: Props) {
  if (agents.length === 0) {
    return (
      <div className={styles.empty}>
        <p>No agents running</p>
        <p className={styles.hint}>Launch an agent from a project board or use the "New Agent" button above.</p>
      </div>
    );
  }

  return (
    <div className={styles.grid}>
      {sortAgents(agents).map((agent) => (
        <AgentCard key={agent.id} agent={agent} onViewLog={onViewLog} onAttach={onAttach} />
      ))}
    </div>
  );
}
