import type { DiffStats as DiffStatsType } from "../../types/api";
import styles from "./DiffStats.module.css";

interface Props {
  stats: DiffStatsType;
  truncated?: boolean;
}

export function DiffStats({ stats, truncated }: Props) {
  return (
    <div className={styles.bar}>
      <span className={styles.files}>{stats.files_changed} file{stats.files_changed !== 1 ? "s" : ""} changed</span>
      {stats.insertions > 0 && (
        <span className={styles.insertions}>+{stats.insertions}</span>
      )}
      {stats.deletions > 0 && (
        <span className={styles.deletions}>-{stats.deletions}</span>
      )}
      {truncated && <span className={styles.truncated}>(truncated)</span>}
    </div>
  );
}
