import { useEffect, useState } from "react";
import type { TraceSummary } from "../types/api";

export function useTraces() {
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

    // Poll every 10s for new traces.
    const interval = setInterval(() => {
      fetch("/-/api/traces")
        .then((r) => r.json())
        .then((data: TraceSummary[]) => setTraces(data ?? []))
        .catch(() => {});
    }, 10000);

    return () => clearInterval(interval);
  }, []);

  return { traces, loading };
}
