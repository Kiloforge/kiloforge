import { useState } from "react";
import type { FileDiff as FileDiffType } from "../../types/api";
import styles from "./FileDiff.module.css";

interface Props {
  file: FileDiffType;
}

const statusStyles: Record<string, { label: string; cls: string }> = {
  added: { label: "A", cls: styles.statusAdded },
  modified: { label: "M", cls: styles.statusModified },
  deleted: { label: "D", cls: styles.statusDeleted },
  renamed: { label: "R", cls: styles.statusRenamed },
};

export function FileDiff({ file }: Props) {
  const [open, setOpen] = useState(true);
  const status = statusStyles[file.status] ?? statusStyles.modified;

  return (
    <div className={styles.file}>
      <div className={styles.fileHeader} onClick={() => setOpen((v) => !v)}>
        <span className={`${styles.chevron} ${open ? styles.chevronOpen : ""}`}>&#9654;</span>
        <span className={`${styles.statusBadge} ${status.cls}`}>{status.label}</span>
        <span className={styles.filePath}>
          {file.old_path ? `${file.old_path} → ${file.path}` : file.path}
        </span>
        <span className={styles.fileCounts}>
          {file.insertions > 0 && <span className={styles.ins}>+{file.insertions}</span>}
          {file.deletions > 0 && <span className={styles.del}>-{file.deletions}</span>}
        </span>
      </div>

      {open && (
        <div className={styles.hunks}>
          {file.is_binary ? (
            <div className={styles.binaryLabel}>Binary file — no diff available</div>
          ) : file.hunks.length === 0 ? (
            <div className={styles.emptyLabel}>No changes</div>
          ) : (
            file.hunks.map((hunk, hi) => (
              <div key={hi}>
                <div className={styles.hunkHeader}>{hunk.header}</div>
                {hunk.lines.map((line, li) => {
                  const cls =
                    line.type === "add"
                      ? styles.lineAdd
                      : line.type === "delete"
                        ? styles.lineDelete
                        : styles.lineContext;
                  const prefix = line.type === "add" ? "+" : line.type === "delete" ? "-" : " ";
                  return (
                    <div key={li} className={`${styles.line} ${cls}`}>
                      <span className={styles.lineNo}>{line.old_no ?? ""}</span>
                      <span className={styles.lineNo}>{line.new_no ?? ""}</span>
                      <span className={styles.lineContent}>{prefix}{line.content}</span>
                    </div>
                  );
                })}
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
