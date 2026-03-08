import { useCallback, useState } from "react";
import { useConsent } from "../hooks/useConsent";
import { ConsentDialog } from "./ConsentDialog";
import styles from "./AdminPanel.module.css";

type AdminOperation = "bulk-archive" | "compact-archive" | "report";

interface Props {
  projectSlug?: string;
  running: boolean;
  onStartOperation: (agentId: string) => void;
}

const operations: { key: AdminOperation; label: string }[] = [
  { key: "bulk-archive", label: "Bulk Archive" },
  { key: "compact-archive", label: "Compact Archive" },
  { key: "report", label: "Generate Report" },
];

export function AdminPanel({ projectSlug, running, onStartOperation }: Props) {
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState<AdminOperation | null>(null);
  const consent = useConsent();

  const handleRun = useCallback(
    async (op: AdminOperation) => {
      setError(null);
      setStarting(op);
      try {
        const resp = await fetch("/api/admin/run", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            operation: op,
            ...(projectSlug ? { project: projectSlug } : {}),
          }),
        });
        if (resp.status === 403) {
          consent.requestConsent(() => handleRun(op));
          return;
        }
        if (!resp.ok) {
          const data = (await resp.json()) as { error?: string };
          setError(data.error ?? `Failed (${resp.status})`);
          return;
        }
        const data = (await resp.json()) as { agent_id: string; ws_url: string };
        onStartOperation(data.agent_id);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Request failed");
      } finally {
        setStarting(null);
      }
    },
    [projectSlug, onStartOperation, consent],
  );

  return (
    <div className={styles.panel}>
      <div className={styles.actions}>
        {operations.map((op) => (
          <button
            key={op.key}
            className={styles.opBtn}
            disabled={running || starting !== null}
            onClick={() => handleRun(op.key)}
          >
            {starting === op.key ? "Starting..." : op.label}
          </button>
        ))}
      </div>
      {error && <p className={styles.error}>{error}</p>}
      {consent.showDialog && <ConsentDialog onAccept={consent.accept} onDeny={consent.deny} />}
    </div>
  );
}
