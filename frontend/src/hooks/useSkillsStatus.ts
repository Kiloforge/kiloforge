import { useState, useEffect, useCallback } from "react";
import type { SkillsStatus } from "../types/api";

interface UseSkillsStatusResult {
  status: SkillsStatus | null;
  loading: boolean;
  updating: boolean;
  triggerUpdate: (force?: boolean) => Promise<void>;
  refresh: () => void;
}

export function useSkillsStatus(): UseSkillsStatusResult {
  const [status, setStatus] = useState<SkillsStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);

  const fetchStatus = useCallback(() => {
    fetch("/-/api/skills")
      .then((r) => r.json())
      .then((data: SkillsStatus) => {
        setStatus(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 60_000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  const triggerUpdate = useCallback(async (force = false) => {
    setUpdating(true);
    try {
      const resp = await fetch("/-/api/skills/update", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ force }),
      });
      if (!resp.ok) {
        const err = await resp.json();
        throw new Error(err.error || "Update failed");
      }
      fetchStatus();
    } finally {
      setUpdating(false);
    }
  }, [fetchStatus]);

  return { status, loading, updating, triggerUpdate, refresh: fetchStatus };
}
