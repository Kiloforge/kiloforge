import styles from "./ConfirmDialog.module.css";

interface ConfirmDialogProps {
  title: string;
  message: string;
  confirmLabel?: string;
  confirming?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  title,
  message,
  confirmLabel = "Confirm",
  confirming = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h3 className={styles.title}>{title}</h3>
        <p className={styles.message}>{message}</p>
        <div className={styles.actions}>
          <button
            className={styles.confirmBtn}
            onClick={onConfirm}
            disabled={confirming}
          >
            {confirming ? `${confirmLabel.replace(/e$/, "")}ing...` : confirmLabel}
          </button>
          <button
            className={styles.cancelBtn}
            onClick={onCancel}
            disabled={confirming}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
