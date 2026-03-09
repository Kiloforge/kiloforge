import type { FileDiff } from "../../types/api";
import styles from "./FileList.module.css";

interface Props {
  files: FileDiff[];
  activeIndex: number;
  onSelect: (index: number) => void;
}

const statusBadge: Record<string, { label: string; cls: string }> = {
  added: { label: "A", cls: styles.badgeA },
  modified: { label: "M", cls: styles.badgeM },
  deleted: { label: "D", cls: styles.badgeD },
  renamed: { label: "R", cls: styles.badgeR },
};

export function FileList({ files, activeIndex, onSelect }: Props) {
  return (
    <div className={styles.list}>
      {files.map((f, i) => {
        const badge = statusBadge[f.status] ?? statusBadge.modified;
        return (
          <button
            key={f.path}
            className={`${styles.item} ${i === activeIndex ? styles.active : ""}`}
            onClick={() => onSelect(i)}
            title={f.old_path ? `${f.old_path} → ${f.path}` : f.path}
          >
            <span className={`${styles.badge} ${badge.cls}`}>{badge.label}</span>
            <span className={styles.path}>
              {f.old_path ? `${f.old_path} → ${f.path}` : f.path}
            </span>
            <span className={styles.counts}>
              {f.insertions > 0 && <span className={styles.ins}>+{f.insertions}</span>}
              {f.deletions > 0 && <span className={styles.del}>-{f.deletions}</span>}
            </span>
          </button>
        );
      })}
    </div>
  );
}
