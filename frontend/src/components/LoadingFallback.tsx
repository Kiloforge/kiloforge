import styles from "./LoadingFallback.module.css";

export function LoadingFallback() {
  return (
    <div className={styles.container}>
      <div className={styles.spinner} />
    </div>
  );
}
