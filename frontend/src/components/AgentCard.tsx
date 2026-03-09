import { Link } from "react-router-dom";
import type { Agent } from "../types/api";
import { useTracks } from "../hooks/useTracks";
import { useAgentActions, canStop, canResume, canDelete } from "../hooks/useAgentActions";
import { StatusBadge } from "./StatusBadge";
import { formatUSD, formatTokens, formatUptime } from "../utils/format";
import styles from "./AgentCard.module.css";

interface Props {
  agent: Agent;
  onViewLog: (agentId: string) => void;
  onAttach?: (agentId: string) => void;
}

export function AgentCard({ agent, onViewLog, onAttach }: Props) {
  const refLink = agent.ref || null;
  const { tracks } = useTracks();
  const { stop, resume, del } = useAgentActions();
  const matchedTrack = refLink ? tracks.find((t) => t.id === refLink) : null;
  const projectSlug = matchedTrack?.project ?? null;

  const hasTokens = (agent.input_tokens ?? 0) > 0 || (agent.output_tokens ?? 0) > 0;
  const cacheRead = agent.cache_read_tokens ?? 0;
  const cacheCreate = agent.cache_creation_tokens ?? 0;
  const hasCache = cacheRead > 0 || cacheCreate > 0;

  return (
    <div className={styles.card}>
      <div className={styles.header}>
        <Link to={`/agents/${agent.id}`} className={styles.name}>{agent.name || agent.id}</Link>
        <span className={`${styles.role} ${styles[agent.role] ?? ""}`}>{agent.role}</span>
      </div>
      {agent.name && (
        <div className={styles.idRow}>{agent.id}</div>
      )}
      <div className={styles.header}>
        <StatusBadge status={agent.status} />
        {hasTokens && (
          <span className={styles.tokens}>
            {formatTokens(agent.input_tokens ?? 0)} / {formatTokens(agent.output_tokens ?? 0)}
          </span>
        )}
      </div>
      {hasCache && (
        <div className={styles.cacheRow}>
          cache: {formatTokens(cacheRead)} read
          {cacheCreate > 0 && <> · {formatTokens(cacheCreate)} create</>}
        </div>
      )}
      <div className={styles.meta}>
        {refLink && (
          projectSlug
            ? <Link to={`/projects/${projectSlug}`}>ref: {refLink}</Link>
            : <Link to={`/agents/${agent.id}`}>ref: {refLink}</Link>
        )}
        {agent.model && <span>model: {agent.model}</span>}
        {agent.uptime_seconds != null && <span>uptime: {formatUptime(agent.uptime_seconds)}</span>}
        {agent.pid > 0 && <span>PID: {agent.pid}</span>}
        {agent.estimated_cost_usd != null && (
          <span className={styles.cost}>est. {formatUSD(agent.estimated_cost_usd)}</span>
        )}
      </div>
      <div className={styles.actions}>
        {agent.role === "interactive" && onAttach && (
          <button className={styles.btn} onClick={() => onAttach(agent.id)}>
            Attach
          </button>
        )}
        {agent.worktree_dir && (
          <Link to={`/agents/${agent.id}#diff`} className={styles.btn}>
            View Diff
          </Link>
        )}
        {agent.log_file && (
          <button className={styles.btn} onClick={() => onViewLog(agent.id)}>
            View Log
          </button>
        )}
        {canStop(agent) && (
          <button
            className={`${styles.btn} ${styles.btnDanger}`}
            onClick={() => stop.mutate(agent.id)}
            disabled={stop.isPending}
          >
            {stop.isPending ? "Stopping..." : "Stop"}
          </button>
        )}
        {canResume(agent) && (
          <button
            className={`${styles.btn} ${styles.btnSuccess}`}
            onClick={() => {
              resume.mutate(agent.id, {
                onSuccess: () => onAttach?.(agent.id),
              });
            }}
            disabled={resume.isPending}
          >
            {resume.isPending ? "Resuming..." : "Resume"}
          </button>
        )}
        {canDelete(agent) && (
          <button
            className={`${styles.btn} ${styles.btnDanger}`}
            onClick={() => {
              if (window.confirm(`Delete agent "${agent.name || agent.id}"?`)) {
                del.mutate(agent.id);
              }
            }}
            disabled={del.isPending}
          >
            {del.isPending ? "Deleting..." : "Delete"}
          </button>
        )}
      </div>
    </div>
  );
}
