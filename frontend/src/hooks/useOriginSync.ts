import { useState, useEffect, useCallback } from "react";
import type { SyncStatus, PushRequest, PullResult, PushResult } from "../types/api";

interface UseOriginSyncResult {
  syncStatus: SyncStatus | null;
  loading: boolean;
  pushing: boolean;
  pulling: boolean;
  error: string | null;
  push: (req: PushRequest) => Promise<PushResult | null>;
  pull: (remoteBranch?: string) => Promise<PullResult | null>;
  refresh: () => void;
  clearError: () => void;
}

export function useOriginSync(slug?: string): UseOriginSyncResult {
  const [syncStatus, setSyncStatus] = useState<SyncStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [pushing, setPushing] = useState(false);
  const [pulling, setPulling] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = useCallback(() => {
    if (!slug) return;
    setLoading(true);
    fetch(`/api/projects/${encodeURIComponent(slug)}/sync-status`)
      .then((r) => {
        if (!r.ok) throw new Error(`Status ${r.status}`);
        return r.json();
      })
      .then((data: SyncStatus) => {
        setSyncStatus(data);
        setLoading(false);
      })
      .catch(() => {
        setSyncStatus(null);
        setLoading(false);
      });
  }, [slug]);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  const push = useCallback(
    async (req: PushRequest): Promise<PushResult | null> => {
      if (!slug) return null;
      setPushing(true);
      setError(null);
      try {
        const resp = await fetch(`/api/projects/${encodeURIComponent(slug)}/push`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        });
        if (!resp.ok) {
          const body = await resp.json().catch(() => ({ error: "Push failed" }));
          setError(body.error || `Error ${resp.status}`);
          setPushing(false);
          return null;
        }
        const result: PushResult = await resp.json();
        setPushing(false);
        fetchStatus();
        return result;
      } catch {
        setError("Network error");
        setPushing(false);
        return null;
      }
    },
    [slug, fetchStatus],
  );

  const pull = useCallback(
    async (remoteBranch?: string): Promise<PullResult | null> => {
      if (!slug) return null;
      setPulling(true);
      setError(null);
      try {
        const resp = await fetch(`/api/projects/${encodeURIComponent(slug)}/pull`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ remote_branch: remoteBranch }),
        });
        if (!resp.ok) {
          const body = await resp.json().catch(() => ({ error: "Pull failed" }));
          setError(body.error || `Error ${resp.status}`);
          setPulling(false);
          return null;
        }
        const result: PullResult = await resp.json();
        setPulling(false);
        fetchStatus();
        return result;
      } catch {
        setError("Network error");
        setPulling(false);
        return null;
      }
    },
    [slug, fetchStatus],
  );

  const clearError = useCallback(() => setError(null), []);

  return { syncStatus, loading, pushing, pulling, error, push, pull, refresh: fetchStatus, clearError };
}
