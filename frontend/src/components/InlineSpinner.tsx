import styles from "./InlineSpinner.module.css";

interface InlineSpinnerProps {
  label?: string;
}

export function InlineSpinner({ label = "Loading..." }: InlineSpinnerProps) {
  return (
    <span className={styles.container} role="status">
      <span className={styles.spinner} />
      <span className={styles.srOnly}>{label}</span>
    </span>
  );
}
