import type { ConnectionState } from "../hooks/useSSE";
import styles from "./ConnectionStatus.module.css";

export function ConnectionStatus({ state }: { state: ConnectionState }) {
  return <span className={`${styles.badge} ${styles[state]}`} data-testid="sse-status" data-status={state}>{state}</span>;
}
