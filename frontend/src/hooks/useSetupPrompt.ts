import { useState, useCallback, useRef } from "react";

/**
 * Hook for prompting kiloforge setup when a 428 is received.
 * Mirrors the useConsent/useSkillsPrompt pattern: call `requestSetup(slug, retryFn)`
 * to show the dialog. After setup completes, retryFn is called automatically.
 */
export function useSetupPrompt() {
  const [showDialog, setShowDialog] = useState(false);
  const [projectSlug, setProjectSlug] = useState("");
  const [agentId, setAgentId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const retryRef = useRef<(() => void) | null>(null);

  const requestSetup = useCallback((slug: string, retry: () => void) => {
    setProjectSlug(slug);
    retryRef.current = retry;
    setError(null);
    setAgentId(null);
    setShowDialog(true);
  }, []);

  const startSetup = useCallback(async () => {
    setError(null);
    setStarting(true);
    try {
      const res = await fetch(`/api/projects/${encodeURIComponent(projectSlug)}/setup`, {
        method: "POST",
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: `Error ${res.status}` }));
        throw new Error(body.error || `Setup failed with status ${res.status}`);
      }
      const data = await res.json();
      setAgentId(data.agent_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to start setup");
    } finally {
      setStarting(false);
    }
  }, [projectSlug]);

  const handleSetupComplete = useCallback(() => {
    setShowDialog(false);
    setAgentId(null);
    const fn = retryRef.current;
    retryRef.current = null;
    fn?.();
  }, []);

  const cancel = useCallback(() => {
    setShowDialog(false);
    setError(null);
    setAgentId(null);
    retryRef.current = null;
  }, []);

  return {
    showDialog,
    projectSlug,
    agentId,
    error,
    starting,
    requestSetup,
    startSetup,
    handleSetupComplete,
    cancel,
  };
}
