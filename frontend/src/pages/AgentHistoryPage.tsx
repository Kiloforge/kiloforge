import { useState, useMemo } from "react";
import { Link } from "react-router-dom";
import type { Agent } from "../types/api";
import { useAgents } from "../hooks/useAgents";
import { useAgentActions, canStop, canResume, canDelete } from "../hooks/useAgentActions";
import { StatusBadge } from "../components/StatusBadge";
import { PaginatedList } from "../components/PaginatedList";
import { InlineSpinner } from "../components/InlineSpinner";
import { formatUSD, formatUptime } from "../utils/format";
import styles from "./AgentHistoryPage.module.css";
import appStyles from "../App.module.css";

const STATUS_OPTIONS = ["all", "running", "waiting", "completed", "failed", "stopped", "suspended"] as const;

export function AgentHistoryPage() {
  const { agents, loading, remainingCount, hasNextPage, isFetchingNextPage, fetchNextPage } = useAgents(false);
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const filtered = useMemo(() => {
    if (statusFilter === "all") return agents;
    if (statusFilter === "running") {
      return agents.filter((a) => a.status === "running" || a.status === "waiting");
    }
    return agents.filter((a) => a.status === statusFilter);
  }, [agents, statusFilter]);

  const sorted = useMemo(() => {
    return [...filtered].sort((a, b) =>
      (b.updated_at || "").localeCompare(a.updated_at || ""),
    );
  }, [filtered]);

  return (
    <div className={styles.page}>
      <div className={styles.topBar}>
        <Link to="/" className={styles.back}>&larr; Back</Link>
        <h2 className={styles.title}>All Agents</h2>
      </div>

      <div className={styles.filterRow}>
        {STATUS_OPTIONS.map((s) => (
          <button
            key={s}
            className={`${styles.chip} ${statusFilter === s ? styles.chipActive : ""}`}
            onClick={() => setStatusFilter(s)}
          >
            {s === "all" ? "All" : s}
          </button>
        ))}
      </div>

      {loading ? (
        <InlineSpinner label="Loading agents..." />
      ) : sorted.length === 0 ? (
        <p className={appStyles.empty}>No agents found</p>
      ) : (
        <PaginatedList
          remainingCount={remainingCount}
          hasNextPage={hasNextPage}
          isFetchingNextPage={isFetchingNextPage}
          onLoadMore={() => fetchNextPage()}
        >
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Name / ID</th>
                  <th>Role</th>
                  <th>Status</th>
                  <th>Ref</th>
                  <th>Uptime</th>
                  <th>Cost</th>
                  <th>Updated</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((agent) => (
                  <AgentRow key={agent.id} agent={agent} />
                ))}
              </tbody>
            </table>
          </div>
        </PaginatedList>
      )}
    </div>
  );
}

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function AgentRow({ agent }: { agent: Agent }) {
  const { stop, resume, del } = useAgentActions();

  return (
    <tr className={styles.row}>
      <td>
        <Link to={`/agents/${agent.id}`} className={styles.agentLink}>
          {agent.name || agent.id}
        </Link>
      </td>
      <td>
        <span className={styles.roleBadge}>{agent.role}</span>
      </td>
      <td>
        <StatusBadge status={agent.status} />
      </td>
      <td className={styles.mono}>{agent.ref || "-"}</td>
      <td>{agent.uptime_seconds != null ? formatUptime(agent.uptime_seconds) : "-"}</td>
      <td>{agent.estimated_cost_usd != null ? formatUSD(agent.estimated_cost_usd) : "-"}</td>
      <td className={styles.dim}>{agent.updated_at ? formatRelativeTime(agent.updated_at) : "-"}</td>
      <td className={styles.rowActions}>
        {canStop(agent) && (
          <button
            className={`${styles.rowBtn} ${styles.rowBtnDanger}`}
            onClick={() => stop.mutate(agent.id)}
            disabled={stop.isPending}
          >
            Stop
          </button>
        )}
        {canResume(agent) && (
          <button
            className={`${styles.rowBtn} ${styles.rowBtnSuccess}`}
            onClick={() => resume.mutate(agent.id)}
            disabled={resume.isPending}
          >
            Resume
          </button>
        )}
        {canDelete(agent) && (
          <button
            className={`${styles.rowBtn} ${styles.rowBtnDanger}`}
            onClick={() => {
              if (window.confirm(`Delete agent "${agent.name || agent.id}"?`)) {
                del.mutate(agent.id);
              }
            }}
            disabled={del.isPending}
          >
            Delete
          </button>
        )}
      </td>
    </tr>
  );
}
