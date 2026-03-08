import { useToast } from "./ToastProvider";
import styles from "./Toast.module.css";

const VARIANT_ICONS: Record<string, string> = {
  error: "\u2716",
  warning: "\u26A0",
  success: "\u2714",
};

export function ToastContainer() {
  const { toasts, removeToast } = useToast();

  if (toasts.length === 0) return null;

  return (
    <div className={styles.container}>
      {toasts.map((toast) => (
        <div key={toast.id} className={`${styles.toast} ${styles[toast.variant]}`}>
          <span className={styles.icon}>{VARIANT_ICONS[toast.variant]}</span>
          <div className={styles.body}>
            <span className={styles.message}>{toast.message}</span>
            {toast.detail && <span className={styles.detail}>{toast.detail}</span>}
          </div>
          <button
            className={styles.dismiss}
            onClick={() => removeToast(toast.id)}
            title="Dismiss"
          >
            &times;
          </button>
        </div>
      ))}
    </div>
  );
}
