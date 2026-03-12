import { useState, useCallback } from "react";
import { AgentTerminal } from "./AgentTerminal";
import { MiniCard } from "./MiniCard";
import styles from "./ConsentDialog.module.css";

interface Props {
  projectSlug: string;
  agentId: string | null;
  agentName?: string;
  agentRole?: string;
  starting: boolean;
  error: string | null;
  onRunSetup: () => void;
  onSetupComplete: () => void;
  onCancel: () => void;
}

export function SetupRequiredDialog({
  projectSlug,
  agentId,
  agentName,
  agentRole,
  starting,
  error,
  onRunSetup,
  onSetupComplete,
  onCancel,
}: Props) {
  const [minimized, setMinimized] = useState(false);

  const handleMinimize = useCallback(() => setMinimized(true), []);
  const handleRestore = useCallback(() => setMinimized(false), []);

  if (agentId) {
    return (
      <>
        <AgentTerminal
          agentId={agentId}
          name={agentName}
          role={agentRole}
          minimized={minimized}
          onMinimize={handleMinimize}
          onClose={onSetupComplete}
        />
        {minimized && (
          <MiniCard
            agentId={agentId}
            name={agentName}
            role={agentRole}
            unreadCount={0}
            notificationType={null}
            initialX={Math.max(8, (window.innerWidth - 200) / 2)}
            initialY={window.innerHeight - 64}
            onRestore={handleRestore}
            onClose={onSetupComplete}
          />
        )}
      </>
    );
  }

  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h3 className={styles.title}>Kiloforge Setup Required</h3>
        <p className={styles.message}>
          Project <strong>{projectSlug}</strong> needs to be set up before you
          can generate tracks or spawn agents. This will run the interactive
          setup wizard to configure conductor artifacts.
        </p>
        {error && <p className={styles.warning}>{error}</p>}
        <div className={styles.actions}>
          <button
            className={styles.acceptBtn}
            onClick={onRunSetup}
            disabled={starting}
          >
            {starting ? "Starting..." : "Run Setup"}
          </button>
          <button
            className={styles.cancelBtn}
            onClick={onCancel}
            disabled={starting}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
