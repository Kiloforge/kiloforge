import { useState, useCallback } from "react";
import styles from "./RemoveProjectDialog.module.css";

interface RemoveProjectDialogProps {
  slug: string;
  removing: boolean;
  onConfirm: (slug: string, cleanup: boolean) => Promise<boolean>;
  onCancel: () => void;
}

export function RemoveProjectDialog({ slug, removing, onConfirm, onCancel }: RemoveProjectDialogProps) {
  const [cleanup, setCleanup] = useState(false);

  const handleConfirm = useCallback(async () => {
    const ok = await onConfirm(slug, cleanup);
    if (ok) onCancel();
  }, [slug, cleanup, onConfirm, onCancel]);

  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h3 className={styles.title}>Remove Project</h3>
        <p className={styles.message}>
          Remove <strong>{slug}</strong>? This will deregister the project from kiloforge.
        </p>
        <label className={styles.checkboxLabel}>
          <input
            type="checkbox"
            checked={cleanup}
            onChange={(e) => setCleanup(e.target.checked)}
            disabled={removing}
          />
          <span>Also delete Gitea repo and local clone</span>
        </label>
        {cleanup && (
          <p className={styles.warning}>
            This will permanently delete the Gitea repository and the local clone directory.
          </p>
        )}
        <div className={styles.actions}>
          <button
            className={styles.removeBtn}
            onClick={handleConfirm}
            disabled={removing}
          >
            {removing ? "Removing..." : "Remove"}
          </button>
          <button
            className={styles.cancelBtn}
            onClick={onCancel}
            disabled={removing}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
