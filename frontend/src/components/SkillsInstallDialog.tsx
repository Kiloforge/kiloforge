import styles from "./ConsentDialog.module.css";

interface Props {
  updating: boolean;
  error: string | null;
  onInstall: () => void;
  onCancel: () => void;
}

export function SkillsInstallDialog({ updating, error, onInstall, onCancel }: Props) {
  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h3 className={styles.title}>Skills Required</h3>
        <p className={styles.message}>
          This action requires skills that are not yet installed.
          Would you like to install them now?
        </p>
        {error && (
          <p className={styles.warning}>{error}</p>
        )}
        <div className={styles.actions}>
          <button
            className={styles.acceptBtn}
            onClick={onInstall}
            disabled={updating}
          >
            {updating ? "Installing..." : "Install Skills"}
          </button>
          <button
            className={styles.cancelBtn}
            onClick={onCancel}
            disabled={updating}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
