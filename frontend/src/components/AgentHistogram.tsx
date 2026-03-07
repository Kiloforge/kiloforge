import type { Agent } from "../types/api";
import styles from "./AgentHistogram.module.css";

export function AgentHistogram({ agents }: { agents: Agent[] }) {
  const counts: Record<string, number> = {};
  for (const agent of agents) {
    counts[agent.status] = (counts[agent.status] ?? 0) + 1;
  }

  const entries = Object.entries(counts);
  if (entries.length === 0) return null;

  return (
    <div className={styles.histogram}>
      {entries.map(([status, count]) => (
        <span key={status} className={`${styles.chip} ${styles[status] ?? ""}`}>
          {count} {status}
        </span>
      ))}
    </div>
  );
}
