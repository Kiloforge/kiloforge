import { useCallback, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useConsent } from "../hooks/useConsent";
import { fetcher, FetchError } from "../api/fetcher";
import { ConsentDialog } from "./ConsentDialog";
import styles from "./AdminPanel.module.css";

type AdminOperation = "bulk-archive" | "compact-archive" | "report";

interface Props {
  projectSlug?: string;
  running: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onStartOperation: (agentId: string) => void;
  onSetupRequired?: () => void;
  onSkillsRequired?: () => void;
}

const operations: { key: AdminOperation; label: string }[] = [
  { key: "bulk-archive", label: "Bulk Archive" },
  { key: "compact-archive", label: "Compact Archive" },
  { key: "report", label: "Generate Report" },
];

export function AdminPanel({ projectSlug, running, disabled, disabledReason, onStartOperation, onSetupRequired, onSkillsRequired }: Props) {
  const [error, setError] = useState<string | null>(null);
  const consent = useConsent();

  const mutation = useMutation({
    mutationFn: (op: AdminOperation) =>
      fetcher<{ agent_id: string; ws_url: string }>("/api/admin/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          operation: op,
          ...(projectSlug ? { project: projectSlug } : {}),
        }),
      }),
    onSuccess: (data) => {
      onStartOperation(data.agent_id);
    },
    onError: (err, op) => {
      if (err instanceof FetchError && err.status === 403) {
        consent.requestConsent(() => handleRun(op));
        return;
      }
      if (err instanceof FetchError && err.status === 412 && onSkillsRequired) {
        onSkillsRequired();
        return;
      }
      if (err instanceof FetchError && err.status === 428 && onSetupRequired) {
        onSetupRequired();
        return;
      }
      if (err instanceof FetchError) {
        const body = err.body as { error?: string };
        setError(body?.error ?? `Failed (${err.status})`);
      } else {
        setError(err instanceof Error ? err.message : "Request failed");
      }
    },
  });

  const handleRun = useCallback(
    (op: AdminOperation) => {
      setError(null);
      mutation.mutate(op);
    },
    [mutation, consent],
  );

  const isDisabled = disabled || running || mutation.isPending;

  return (
    <div className={styles.panel}>
      <div className={styles.actions}>
        {operations.map((op) => (
          <button
            key={op.key}
            className={styles.opBtn}
            disabled={isDisabled}
            title={disabled ? disabledReason : undefined}
            onClick={() => handleRun(op.key)}
          >
            {mutation.isPending && mutation.variables === op.key ? "Starting..." : op.label}
          </button>
        ))}
      </div>
      {error && <p className={styles.error}>{error}</p>}
      {consent.showDialog && <ConsentDialog onAccept={consent.accept} onDeny={consent.deny} />}
    </div>
  );
}
