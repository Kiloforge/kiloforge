import styles from "./ThinkingIndicator.module.css";

interface ThinkingIndicatorProps {
  label?: string;
}

export function ThinkingIndicator({ label = "Thinking" }: ThinkingIndicatorProps) {
  return (
    <span className={styles.container} role="status" aria-live="polite">
      <span className={styles.label}>{label}</span>
      <span className={styles.dot} />
      <span className={styles.dot} />
      <span className={styles.dot} />
    </span>
  );
}
