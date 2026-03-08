import { useState, useCallback, useRef } from "react";

/**
 * Hook for managing agent-permissions consent flow.
 * When an agent-spawn endpoint returns 403, call `requestConsent(retryFn)`
 * to show the consent dialog. On accept, consent is recorded and retryFn is called.
 */
export function useConsent() {
  const [showDialog, setShowDialog] = useState(false);
  const retryRef = useRef<(() => void) | null>(null);

  const requestConsent = useCallback((retry: () => void) => {
    retryRef.current = retry;
    setShowDialog(true);
  }, []);

  const accept = useCallback(async () => {
    await fetch("/api/consent/agent-permissions", { method: "POST" });
    setShowDialog(false);
    const fn = retryRef.current;
    retryRef.current = null;
    fn?.();
  }, []);

  const deny = useCallback(() => {
    setShowDialog(false);
    retryRef.current = null;
  }, []);

  return { showDialog, requestConsent, accept, deny };
}
