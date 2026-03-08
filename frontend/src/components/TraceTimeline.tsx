import { useMemo } from "react";
import type { SpanInfo } from "../types/api";
import styles from "./TraceTimeline.module.css";

interface Props {
  spans: SpanInfo[];
  onSpanClick?: (span: SpanInfo) => void;
}

export function TraceTimeline({ spans, onSpanClick }: Props) {
  const { rows } = useMemo(() => {
    if (spans.length === 0) return { rows: [], minTime: 0, totalDuration: 1 };

    const sorted = [...spans].sort(
      (a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
    );

    const min = new Date(sorted[0].start_time).getTime();
    const max = Math.max(...sorted.map((s) => new Date(s.end_time).getTime()));
    const duration = max - min || 1;

    // Build parent-child tree for indentation.
    const depthMap = new Map<string, number>();
    for (const s of sorted) {
      if (!s.parent_id) {
        depthMap.set(s.span_id, 0);
      } else {
        depthMap.set(s.span_id, (depthMap.get(s.parent_id) ?? 0) + 1);
      }
    }

    return {
      rows: sorted.map((s) => ({
        span: s,
        depth: depthMap.get(s.span_id) ?? 0,
        leftPct: ((new Date(s.start_time).getTime() - min) / duration) * 100,
        widthPct: Math.max((s.duration_ms / duration) * 100, 0.5),
      })),
      minTime: min,
      totalDuration: duration,
    };
  }, [spans]);

  if (spans.length === 0) {
    return <div className={styles.empty}>No spans recorded</div>;
  }

  return (
    <div className={styles.timeline}>
      <div className={styles.header}>
        <span className={styles.nameCol}>Span</span>
        <span className={styles.durationCol}>Duration</span>
        <span className={styles.barCol}>Timeline</span>
      </div>
      {rows.map(({ span, depth, leftPct, widthPct }) => (
        <div
          key={span.span_id}
          className={styles.row}
          onClick={() => onSpanClick?.(span)}
        >
          <span
            className={styles.nameCol}
            style={{ paddingLeft: `${depth * 16 + 4}px` }}
          >
            {span.name}
          </span>
          <span className={styles.durationCol}>{span.duration_ms}ms</span>
          <span className={styles.barCol}>
            <span
              className={`${styles.bar} ${styles[span.status] ?? styles.ok}`}
              style={{ left: `${leftPct}%`, width: `${widthPct}%` }}
            />
          </span>
        </div>
      ))}
    </div>
  );
}
