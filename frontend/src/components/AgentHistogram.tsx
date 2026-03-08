import type { Agent } from "../types/api";
import styles from "./AgentHistogram.module.css";

export function AgentHistogram({ agents }: { agents: Agent[] }) {
  const active = agents.filter(
    (a) => a.status === "running" || a.status === "waiting",
  );
  const counts: Record<string, number> = {};
  for (const agent of active) {
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
