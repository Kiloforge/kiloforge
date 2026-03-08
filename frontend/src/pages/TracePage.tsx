import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import type { TraceDetail, SpanInfo } from "../types/api";
import { TraceTimeline } from "../components/TraceTimeline";

export function TracePage() {
  const { traceId } = useParams<{ traceId: string }>();
  const [trace, setTrace] = useState<TraceDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<SpanInfo | null>(null);

  useEffect(() => {
    if (!traceId) return;
    fetch(`/api/traces/${traceId}`)
      .then((r) => {
        if (!r.ok) throw new Error("Trace not found");
        return r.json();
      })
      .then((data: TraceDetail) => setTrace(data))
      .catch((err) => setError(err.message));
  }, [traceId]);

  if (error) {
    return (
      <div style={{ padding: 24 }}>
        <Link to="/">&larr; Back</Link>
        <p style={{ color: "#f44336" }}>{error}</p>
      </div>
    );
  }

  if (!trace) {
    return <div style={{ padding: 24 }}>Loading trace...</div>;
  }

  const totalDuration = trace.spans.length > 0
    ? Math.max(...trace.spans.map((s) => new Date(s.end_time).getTime())) -
      Math.min(...trace.spans.map((s) => new Date(s.start_time).getTime()))
    : 0;

  return (
    <div style={{ padding: "16px 24px" }}>
      <div style={{ marginBottom: 16 }}>
        <Link to="/" style={{ color: "#888" }}>&larr; Dashboard</Link>
      </div>

      <h2 style={{ margin: "0 0 4px" }}>
        Trace: {trace.spans.find((s) => !s.parent_id)?.name ?? traceId}
      </h2>
      <p style={{ color: "#888", margin: "0 0 16px", fontFamily: "monospace", fontSize: 12 }}>
        {traceId} &middot; {trace.spans.length} span{trace.spans.length !== 1 ? "s" : ""} &middot;{" "}
        {totalDuration}ms total
      </p>

      <TraceTimeline spans={trace.spans} onSpanClick={setSelected} />

      {selected && (
        <div
          style={{
            marginTop: 16,
            padding: 16,
            background: "#1a1a2e",
            borderRadius: 6,
            fontFamily: "monospace",
            fontSize: 13,
          }}
        >
          <h3 style={{ margin: "0 0 8px" }}>{selected.name}</h3>
          <p style={{ color: "#888", margin: "0 0 8px" }}>
            {selected.duration_ms}ms &middot; {selected.status}
          </p>
          {selected.attributes && Object.keys(selected.attributes).length > 0 && (
            <div style={{ marginBottom: 8 }}>
              <strong style={{ color: "#888" }}>Attributes:</strong>
              <table style={{ marginTop: 4 }}>
                <tbody>
                  {Object.entries(selected.attributes).map(([k, v]) => (
                    <tr key={k}>
                      <td style={{ color: "#4caf50", paddingRight: 12 }}>{k}</td>
                      <td style={{ color: "#ccc" }}>
                        {k === "session.id" ? (
                          <Link
                            to={`/`}
                            title={`Session: ${v}`}
                            style={{ color: "#64b5f6", textDecoration: "none" }}
                          >
                            {v}
                          </Link>
                        ) : k === "track.id" ? (
                          <span
                            style={{ color: "#4caf50", cursor: "default" }}
                            title={`Track: ${v}`}
                          >
                            {v}
                          </span>
                        ) : (
                          v
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {selected.events && selected.events.length > 0 && (
            <div>
              <strong style={{ color: "#888" }}>Events:</strong>
              <ul style={{ margin: "4px 0 0", paddingLeft: 16 }}>
                {selected.events.map((ev, i) => (
                  <li key={i} style={{ color: "#ccc" }}>
                    {ev.name}
                    <span style={{ color: "#666", marginLeft: 8 }}>
                      {new Date(ev.timestamp).toLocaleTimeString()}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
