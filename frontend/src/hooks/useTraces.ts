import { useEffect, useState, useCallback } from "react";
import type { TraceSummary, SSEEventData } from "../types/api";

interface UseTracesResult {
  traces: TraceSummary[];
  loading: boolean;
  handleTraceUpdate: (raw: unknown) => void;
}

export function useTraces(): UseTracesResult {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/-/api/traces")
      .then((r) => r.json())
      .then((data: TraceSummary[]) => {
        setTraces(data ?? []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleTraceUpdate = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const data = event.data as TraceSummary;
    if (!data?.trace_id) return;
    setTraces((prev) => {
      const idx = prev.findIndex((t) => t.trace_id === data.trace_id);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = { ...next[idx], ...data };
        return next;
      }
      return [data, ...prev];
    });
  }, []);

  return { traces, loading, handleTraceUpdate };
}
