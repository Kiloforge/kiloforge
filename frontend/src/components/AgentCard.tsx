import type { Agent } from "../types/api";
import { StatusBadge } from "./StatusBadge";
import { formatUSD, formatUptime } from "../utils/format";
import styles from "./AgentCard.module.css";

interface Props {
  agent: Agent;
  giteaURL: string;
  onViewLog: (agentId: string) => void;
}

export function AgentCard({ agent, giteaURL, onViewLog }: Props) {
  const prMatch = agent.ref?.match(/^PR #(\d+)$/);
  const refLink =
    prMatch && giteaURL ? (
      <a href={`${giteaURL}/-/pulls/${prMatch[1]}`} target="_blank" rel="noopener noreferrer">
        {agent.ref}
      </a>
    ) : (
      agent.ref || null
    );

  return (
    <div className={styles.card}>
      <div className={styles.header}>
        <span className={styles.id}>{agent.id}</span>
        <span className={`${styles.role} ${styles[agent.role] ?? ""}`}>{agent.role}</span>
      </div>
      <div className={styles.header}>
        <StatusBadge status={agent.status} />
        {agent.cost_usd != null && (
          <span className={styles.cost}>{formatUSD(agent.cost_usd)}</span>
        )}
      </div>
      <div className={styles.meta}>
        {refLink && <span>ref: {refLink}</span>}
        {agent.model && <span>model: {agent.model}</span>}
        {agent.uptime_seconds != null && <span>uptime: {formatUptime(agent.uptime_seconds)}</span>}
        {agent.pid > 0 && <span>PID: {agent.pid}</span>}
      </div>
      <div className={styles.actions}>
        {agent.log_file && (
          <button className={styles.btn} onClick={() => onViewLog(agent.id)}>
            View Log
          </button>
        )}
      </div>
    </div>
  );
}
