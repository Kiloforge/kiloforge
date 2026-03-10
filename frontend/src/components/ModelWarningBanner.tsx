import { useState } from "react";
import styles from "./ModelWarningBanner.module.css";

const STORAGE_KEY = "kf_model_warning_dismissed";

export function ModelWarningBanner() {
  const [dismissed, setDismissed] = useState(
    () => localStorage.getItem(STORAGE_KEY) === "1",
  );

  if (dismissed) return null;

  const handleDismiss = () => {
    localStorage.setItem(STORAGE_KEY, "1");
    setDismissed(true);
  };

  return (
    <div className={styles.banner}>
      <span className={styles.message}>
        <strong>Notice:</strong> Kiloforge requires Claude Code and has only been
        thoroughly tested with <strong>Claude Opus 4.6</strong>. Use with other
        models is at your own risk — there may be a higher potential for AI agent
        hallucinations.
      </span>
      <button className={styles.btn} onClick={handleDismiss}>
        Understood
      </button>
    </div>
  );
}
