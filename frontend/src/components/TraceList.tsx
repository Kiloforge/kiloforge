import { Link } from "react-router-dom";
import type { TraceSummary } from "../types/api";

interface Props {
  traces: TraceSummary[];
}

export function TraceList({ traces }: Props) {
  if (traces.length === 0) {
    return (
      <p style={{ color: "#666", padding: "8px 0" }}>
        No traces recorded. Enable tracing with <code>"tracing_enabled": true</code> in config.
      </p>
    );
  }

  return (
    <table style={{ width: "100%", borderCollapse: "collapse", fontFamily: "monospace", fontSize: 13 }}>
      <thead>
        <tr style={{ color: "#888", textAlign: "left", borderBottom: "1px solid #333" }}>
          <th style={{ padding: "6px 8px" }}>Root Span</th>
          <th style={{ padding: "6px 8px" }}>Spans</th>
          <th style={{ padding: "6px 8px" }}>Duration</th>
          <th style={{ padding: "6px 8px" }}>Started</th>
        </tr>
      </thead>
      <tbody>
        {traces.map((t) => {
          const duration =
            new Date(t.end_time).getTime() - new Date(t.start_time).getTime();
          return (
            <tr key={t.trace_id} style={{ borderBottom: "1px solid #222" }}>
              <td style={{ padding: "6px 8px" }}>
                <Link
                  to={`/traces/${t.trace_id}`}
                  style={{ color: "#4caf50", textDecoration: "none" }}
                >
                  {t.root_name || t.trace_id.slice(0, 12)}
                </Link>
              </td>
              <td style={{ padding: "6px 8px", color: "#888" }}>{t.span_count}</td>
              <td style={{ padding: "6px 8px", color: "#888" }}>{duration}ms</td>
              <td style={{ padding: "6px 8px", color: "#666" }}>
                {new Date(t.start_time).toLocaleTimeString()}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
