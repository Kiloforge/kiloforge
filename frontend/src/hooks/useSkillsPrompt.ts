import { useState, useCallback, useRef } from "react";
import { useSkillsStatus } from "./useSkillsStatus";

/**
 * Hook for prompting skills installation when a 412 is received.
 * Mirrors the useConsent pattern: call `requestInstall(retryFn)` to show
 * the dialog. On successful install, retryFn is called automatically.
 */
export function useSkillsPrompt() {
  const [showDialog, setShowDialog] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const retryRef = useRef<(() => void) | null>(null);
  const { updating, triggerUpdate } = useSkillsStatus();

  const requestInstall = useCallback((retry: () => void) => {
    retryRef.current = retry;
    setError(null);
    setShowDialog(true);
  }, []);

  const install = useCallback(async () => {
    setError(null);
    try {
      await triggerUpdate(true);
      setShowDialog(false);
      const fn = retryRef.current;
      retryRef.current = null;
      fn?.();
    } catch {
      setError("Failed to install skills. Please try again.");
    }
  }, [triggerUpdate]);

  const cancel = useCallback(() => {
    setShowDialog(false);
    setError(null);
    retryRef.current = null;
  }, []);

  return { showDialog, updating, error, requestInstall, install, cancel };
}
