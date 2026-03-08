import { Link } from "react-router-dom";
import type { TraceSummary, ConfigResponse, UpdateConfigRequest } from "../types/api";
import { TracingToggle } from "./TracingToggle";

interface Props {
  traces: TraceSummary[];
  config: ConfigResponse | null;
  configLoading: boolean;
  configUpdating: boolean;
  onUpdateConfig: (req: UpdateConfigRequest) => Promise<boolean>;
}

export function TraceList({ traces, config, configLoading, configUpdating, onUpdateConfig }: Props) {
  if (traces.length === 0) {
    return (
      <div style={{ padding: "8px 0" }}>
        <TracingToggle config={config} loading={configLoading} updating={configUpdating} onUpdate={onUpdateConfig} />
        <p style={{ color: "#666", marginTop: 12 }}>
          No traces recorded.{" "}
          {config && !config.tracing_enabled
            ? "Enable tracing above to start collecting traces."
            : "Traces will appear here once operations are recorded."}
        </p>
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 12 }}>
        <TracingToggle config={config} loading={configLoading} updating={configUpdating} onUpdate={onUpdateConfig} />
      </div>
    <table style={{ width: "100%", borderCollapse: "collapse", fontFamily: "monospace", fontSize: 13 }}>
      <thead>
        <tr style={{ color: "#888", textAlign: "left", borderBottom: "1px solid #333" }}>
          <th style={{ padding: "6px 8px" }}>Root Span</th>
          <th style={{ padding: "6px 8px" }}>Track</th>
          <th style={{ padding: "6px 8px" }}>Spans</th>
          <th style={{ padding: "6px 8px" }}>Duration</th>
          <th style={{ padding: "6px 8px" }}>Started</th>
        </tr>
      </thead>
      <tbody>
        {traces.map((t) => {
          const duration =
            new Date(t.end_time).getTime() - new Date(t.start_time).getTime();
          const trackId = t.root_name?.startsWith("track/")
            ? t.root_name.slice(6)
            : null;
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
              <td style={{ padding: "6px 8px", color: "#888", fontSize: 11 }}>
                {trackId || "\u2014"}
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
    </div>
  );
}
