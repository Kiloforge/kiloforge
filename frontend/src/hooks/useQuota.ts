import { useState, useEffect, useCallback } from "react";
import type { QuotaResponse, SSEEventData } from "../types/api";

interface UseQuotaResult {
  quota: QuotaResponse | null;
  loading: boolean;
  handleQuotaUpdate: (raw: unknown) => void;
}

export function useQuota(): UseQuotaResult {
  const [quota, setQuota] = useState<QuotaResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/quota")
      .then((r) => r.json())
      .then((data: QuotaResponse) => {
        setQuota(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleQuotaUpdate = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    setQuota(event.data as QuotaResponse);
  }, []);

  return { quota, loading, handleQuotaUpdate };
}
