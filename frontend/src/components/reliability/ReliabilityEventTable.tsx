import { useState, useMemo } from "react";
import type { ReliabilityEvent } from "../../types/api";
import { useReliabilityEvents } from "../../hooks/useReliability";
import { PaginatedList } from "../PaginatedList";
import styles from "./ReliabilityEventTable.module.css";

const EVENT_TYPES = [
  "all",
  "lock_contention",
  "lock_timeout",
  "agent_timeout",
  "agent_spawn_failure",
  "agent_resume_failure",
  "merge_conflict",
  "quota_exceeded",
] as const;

const SEVERITIES = ["all", "warn", "error", "critical"] as const;

const TYPE_LABELS: Record<string, string> = {
  lock_contention: "lock contention",
  lock_timeout: "lock timeout",
  agent_timeout: "agent timeout",
  agent_spawn_failure: "spawn failure",
  agent_resume_failure: "resume failure",
  merge_conflict: "merge conflict",
  quota_exceeded: "quota exceeded",
};

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

function severityClass(severity: string): string {
  if (severity === "critical") return styles.sevCritical;
  if (severity === "error") return styles.sevError;
  return styles.sevWarn;
}

export function ReliabilityEventTable() {
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [severityFilter, setSeverityFilter] = useState<string>("all");

  const filters = useMemo(() => {
    const f: Record<string, string> = {};
    if (typeFilter !== "all") f.event_type = typeFilter;
    if (severityFilter !== "all") f.severity = severityFilter;
    return f;
  }, [typeFilter, severityFilter]);

  const {
    events,
    isLoading,
    remainingCount,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useReliabilityEvents(Object.keys(filters).length > 0 ? filters : undefined);

  return (
    <div>
      <div className={styles.filterRow}>
        {EVENT_TYPES.map((t) => (
          <button
            key={t}
            className={`${styles.chip} ${typeFilter === t ? styles.chipActive : ""}`}
            onClick={() => setTypeFilter(t)}
          >
            {t === "all" ? "All Types" : TYPE_LABELS[t] ?? t}
          </button>
        ))}
      </div>
      <div className={styles.filterRow}>
        {SEVERITIES.map((s) => (
          <button
            key={s}
            className={`${styles.chip} ${severityFilter === s ? styles.chipActive : ""}`}
            onClick={() => setSeverityFilter(s)}
          >
            {s === "all" ? "All Severities" : s}
          </button>
        ))}
      </div>

      {isLoading ? (
        <p className={styles.dim}>Loading events...</p>
      ) : events.length === 0 ? (
        null // empty state handled by parent
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
                  <th>Time</th>
                  <th>Type</th>
                  <th>Severity</th>
                  <th>Agent</th>
                  <th>Scope</th>
                  <th>Detail</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event) => (
                  <EventRow key={event.id} event={event} />
                ))}
              </tbody>
            </table>
          </div>
        </PaginatedList>
      )}
    </div>
  );
}

function EventRow({ event }: { event: ReliabilityEvent }) {
  return (
    <tr className={styles.row}>
      <td className={styles.dim}>{formatRelativeTime(event.created_at)}</td>
      <td>
        <span className={styles.typeBadge}>
          {TYPE_LABELS[event.event_type] ?? event.event_type}
        </span>
      </td>
      <td>
        <span className={`${styles.sevBadge} ${severityClass(event.severity)}`}>
          {event.severity}
        </span>
      </td>
      <td className={styles.mono}>{event.agent_id ?? "-"}</td>
      <td className={styles.mono}>{event.scope ?? "-"}</td>
      <td className={styles.detail}>
        {event.detail ? JSON.stringify(event.detail) : "-"}
      </td>
    </tr>
  );
}
