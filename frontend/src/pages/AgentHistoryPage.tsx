import { useState, useMemo } from "react";
import { Link } from "react-router-dom";
import type { Agent } from "../types/api";
import { useAgents } from "../hooks/useAgents";
import { StatusBadge } from "../components/StatusBadge";
import { formatUSD, formatUptime } from "../utils/format";
import styles from "./AgentHistoryPage.module.css";
import appStyles from "../App.module.css";

const STATUS_OPTIONS = ["all", "running", "waiting", "completed", "failed", "stopped", "suspended"] as const;

export function AgentHistoryPage() {
  const { agents, loading } = useAgents(false);
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
        <p className={appStyles.empty}>Loading agents...</p>
      ) : sorted.length === 0 ? (
        <p className={appStyles.empty}>No agents found</p>
      ) : (
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
              </tr>
            </thead>
            <tbody>
              {sorted.map((agent) => (
                <AgentRow key={agent.id} agent={agent} />
              ))}
            </tbody>
          </table>
        </div>
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
    </tr>
  );
}
