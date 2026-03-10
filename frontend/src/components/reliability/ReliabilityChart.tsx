import { useState, useMemo } from "react";
import type { ReliabilitySummary } from "../../types/api";
import { useReliabilitySummary } from "../../hooks/useReliability";
import styles from "./ReliabilityChart.module.css";

const TIME_WINDOWS = [
  { label: "1h", hours: 1, bucket: "hour" as const },
  { label: "24h", hours: 24, bucket: "hour" as const },
  { label: "7d", hours: 168, bucket: "day" as const },
  { label: "30d", hours: 720, bucket: "day" as const },
];

// Map severity classes for stacked bars.
const SEVERITY_TYPES: Record<string, string> = {
  lock_contention: "warn",
  lock_timeout: "error",
  agent_timeout: "error",
  agent_spawn_failure: "error",
  agent_resume_failure: "error",
  merge_conflict: "critical",
  quota_exceeded: "critical",
};

function classForSeverity(sev: string): string {
  if (sev === "critical") return styles.barCritical;
  if (sev === "error") return styles.barError;
  return styles.barWarn;
}

interface Props {
  onWindowChange?: (since: string, bucket: string) => void;
}

export function ReliabilityChart({ onWindowChange }: Props) {
  const [windowIdx, setWindowIdx] = useState(1); // default 24h
  const window = TIME_WINDOWS[windowIdx];

  const since = useMemo(() => {
    const d = new Date();
    d.setHours(d.getHours() - window.hours);
    return d.toISOString();
  }, [window.hours]);

  const { summary } = useReliabilitySummary(since, window.bucket);

  const handleWindowChange = (idx: number) => {
    setWindowIdx(idx);
    const w = TIME_WINDOWS[idx];
    const d = new Date();
    d.setHours(d.getHours() - w.hours);
    onWindowChange?.(d.toISOString(), w.bucket);
  };

  return (
    <div className={styles.container}>
      <div className={styles.timeControls}>
        {TIME_WINDOWS.map((w, i) => (
          <button
            key={w.label}
            className={`${styles.timeBtn} ${i === windowIdx ? styles.timeBtnActive : ""}`}
            onClick={() => handleWindowChange(i)}
          >
            {w.label}
          </button>
        ))}
      </div>
      <ChartBars summary={summary} bucket={window.bucket} />
    </div>
  );
}

function ChartBars({ summary, bucket }: { summary: ReliabilitySummary | null; bucket: string }) {
  const buckets = summary?.buckets ?? [];

  if (buckets.length === 0) {
    return <div className={styles.emptyChart}>No events in this time window</div>;
  }

  // Compute max total per bucket for scaling.
  const maxTotal = Math.max(
    1,
    ...buckets.map((b) =>
      Object.values(b.counts).reduce((sum, c) => sum + c, 0),
    ),
  );

  return (
    <div className={styles.chart}>
      {buckets.map((b) => {
        const total = Object.values(b.counts).reduce((sum, c) => sum + c, 0);
        const heightPct = (total / maxTotal) * 100;
        // Group counts by severity.
        const sevCounts: Record<string, number> = { warn: 0, error: 0, critical: 0 };
        for (const [type, count] of Object.entries(b.counts)) {
          const sev = SEVERITY_TYPES[type] ?? "warn";
          sevCounts[sev] += count;
        }

        const dt = new Date(b.timestamp);
        const label =
          bucket === "hour"
            ? `${dt.getHours().toString().padStart(2, "0")}:00`
            : `${(dt.getMonth() + 1)}/${dt.getDate()}`;

        return (
          <div key={b.timestamp} className={styles.barGroup} title={`${total} events`}>
            <div style={{ height: `${heightPct}%`, width: "100%", display: "flex", flexDirection: "column", justifyContent: "flex-end" }}>
              {sevCounts.critical > 0 && (
                <div
                  className={`${styles.bar} ${classForSeverity("critical")}`}
                  style={{ height: `${(sevCounts.critical / total) * 100}%`, minHeight: 2 }}
                />
              )}
              {sevCounts.error > 0 && (
                <div
                  className={`${styles.bar} ${classForSeverity("error")}`}
                  style={{ height: `${(sevCounts.error / total) * 100}%`, minHeight: 2 }}
                />
              )}
              {sevCounts.warn > 0 && (
                <div
                  className={`${styles.bar} ${classForSeverity("warn")}`}
                  style={{ height: `${(sevCounts.warn / total) * 100}%`, minHeight: 2 }}
                />
              )}
            </div>
            <div className={styles.barLabel}>{label}</div>
          </div>
        );
      })}
    </div>
  );
}
