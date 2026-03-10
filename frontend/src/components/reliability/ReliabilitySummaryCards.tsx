import type { ReliabilitySummary } from "../../types/api";
import styles from "./ReliabilitySummaryCards.module.css";

const EVENT_TYPE_LABELS: Record<string, string> = {
  lock_contention: "Lock Contention",
  lock_timeout: "Lock Timeout",
  agent_timeout: "Agent Timeout",
  agent_spawn_failure: "Spawn Failure",
  agent_resume_failure: "Resume Failure",
  merge_conflict: "Merge Conflict",
  quota_exceeded: "Quota Exceeded",
};

// Map event types to a dominant severity for color coding.
const EVENT_SEVERITY: Record<string, "warn" | "error" | "critical"> = {
  lock_contention: "warn",
  lock_timeout: "error",
  agent_timeout: "error",
  agent_spawn_failure: "error",
  agent_resume_failure: "error",
  merge_conflict: "critical",
  quota_exceeded: "critical",
};

interface Props {
  summary: ReliabilitySummary | null;
}

export function ReliabilitySummaryCards({ summary }: Props) {
  const totals = summary?.totals ?? {};
  const entries = Object.entries(totals).filter(([, count]) => count > 0);

  if (entries.length === 0 && summary) {
    return null;
  }

  const allTypes = Object.keys(EVENT_TYPE_LABELS);
  const displayTypes = entries.length > 0
    ? entries.map(([type]) => type)
    : allTypes.slice(0, 4);

  return (
    <div className={styles.cards}>
      {displayTypes.map((type) => {
        const count = totals[type] ?? 0;
        const severity = EVENT_SEVERITY[type] ?? "warn";
        return (
          <div
            key={type}
            className={`${styles.card} ${styles[severity] ?? ""}`}
          >
            <div className={styles.label}>
              {EVENT_TYPE_LABELS[type] ?? type}
            </div>
            <div className={`${styles.count} ${count > 0 ? styles[`${severity}Text`] ?? "" : ""}`}>
              {count}
            </div>
          </div>
        );
      })}
    </div>
  );
}
