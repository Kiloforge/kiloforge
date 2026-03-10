import { useState } from "react";
import styles from "./ConsentDialog.module.css";

interface Props {
  onAccept: () => void;
  onDeny: () => void;
}

export function ConsentDialog({ onAccept, onDeny }: Props) {
  const [accepting, setAccepting] = useState(false);

  const handleAccept = async () => {
    setAccepting(true);
    onAccept();
  };

  return (
    <div className={styles.overlay} onClick={onDeny}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h3 className={styles.title}>Agent Permissions Required</h3>
        <p className={styles.message}>
          Claude Code will run in <strong>Dangerously bypass permissions</strong> mode
          (<code className={styles.code}>--dangerously-skip-permissions</code>).
          This grants agents unrestricted access to tools (file read/write,
          shell commands, network access, etc.) within their worktree directory.
        </p>
        <p className={styles.warning}>
          This is required for non-interactive agent operation.
        </p>
        <div className={styles.actions}>
          <button
            className={styles.acceptBtn}
            onClick={handleAccept}
            disabled={accepting}
          >
            {accepting ? "Accepting..." : "Accept"}
          </button>
          <button
            className={styles.cancelBtn}
            onClick={onDeny}
            disabled={accepting}
          >
            Deny
          </button>
        </div>
      </div>
    </div>
  );
}
