import { useEffect, useCallback } from "react";
import { getShortcutList } from "../hooks/useKeyboardShortcuts";
import { usePlatform } from "../hooks/usePlatform";
import styles from "./ShortcutHelp.module.css";

interface Props {
  onClose: () => void;
}

export function ShortcutHelp({ onClose }: Props) {
  const { isMac } = usePlatform();
  const shortcuts = getShortcutList(isMac);
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      }
    },
    [onClose],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          <h2 className={styles.title}>Keyboard Shortcuts</h2>
          <button className={styles.closeBtn} onClick={onClose}>
            &times;
          </button>
        </div>
        <div className={styles.grid}>
          {shortcuts.map((entry) => (
            <div key={entry.keys} className={styles.row}>
              <kbd className={styles.keys}>{entry.keys}</kbd>
              <span className={styles.desc}>{entry.description}</span>
            </div>
          ))}
        </div>
        {isMac && (
          <p className={styles.hint}>
            Use Ctrl on Windows/Linux instead of &#8984;
          </p>
        )}
      </div>
    </div>
  );
}
