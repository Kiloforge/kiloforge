import { useState, useCallback } from "react";
import type { Agent } from "../types/api";
import { AgentStatusPopover } from "./AgentStatusPopover";
import styles from "./AgentHistogram.module.css";

export function AgentHistogram({ agents }: { agents: Agent[] }) {
  const [openStatus, setOpenStatus] = useState<string | null>(null);

  const handleChipClick = useCallback((status: string) => {
    setOpenStatus((prev) => (prev === status ? null : status));
  }, []);

  const handleClose = useCallback(() => {
    setOpenStatus(null);
  }, []);

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
        <span key={status} className={styles.chipWrapper}>
          <button
            type="button"
            className={`${styles.chip} ${styles[status] ?? ""}`}
            onClick={() => handleChipClick(status)}
          >
            {count} {status}
          </button>
          {openStatus === status && (
            <AgentStatusPopover
              status={status}
              agents={active.filter((a) => a.status === status)}
              onClose={handleClose}
            />
          )}
        </span>
      ))}
    </div>
  );
}
