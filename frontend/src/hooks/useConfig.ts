import { useEffect, useState, useCallback } from "react";
import type { ConfigResponse, UpdateConfigRequest } from "../types/api";

interface UseConfigResult {
  config: ConfigResponse | null;
  loading: boolean;
  updating: boolean;
  updateConfig: (req: UpdateConfigRequest) => Promise<boolean>;
}

export function useConfig(): UseConfigResult {
  const [config, setConfig] = useState<ConfigResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    fetch("/api/config")
      .then((r) => r.json())
      .then((data: ConfigResponse) => {
        setConfig(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const updateConfig = useCallback(
    async (req: UpdateConfigRequest): Promise<boolean> => {
      setUpdating(true);
      try {
        const resp = await fetch("/api/config", {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        });
        if (!resp.ok) {
          setUpdating(false);
          return false;
        }
        const data: ConfigResponse = await resp.json();
        setConfig(data);
        setUpdating(false);
        return true;
      } catch {
        setUpdating(false);
        return false;
      }
    },
    [],
  );

  return { config, loading, updating, updateConfig };
}
