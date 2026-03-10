import { useEffect, useCallback } from "react";
import { useCommandPanes } from "../../hooks/useCommandPanes";
import type { Agent } from "../../types/api";
import { SplitContainer } from "./SplitContainer";
import styles from "./FullScreenCommand.module.css";

interface Props {
  agents: Agent[];
  onExit: () => void;
}

export function FullScreenCommand({ agents, onExit }: Props) {
  const panes = useCommandPanes();

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;

      // Escape exits full-screen mode
      if (e.key === "Escape") {
        e.preventDefault();
        onExit();
        return;
      }

      if (!mod) return;

      // Cmd+D — split vertical
      if (e.key === "d" && !e.shiftKey) {
        e.preventDefault();
        panes.splitPane(panes.activePaneId, "horizontal");
        return;
      }

      // Cmd+Shift+D — split horizontal
      if (e.key === "D" && e.shiftKey) {
        e.preventDefault();
        panes.splitPane(panes.activePaneId, "vertical");
        return;
      }

      // Cmd+W — close active pane (exit if last)
      if (e.key === "w" && !e.shiftKey) {
        e.preventDefault();
        const lastClosed = panes.closePane(panes.activePaneId);
        if (lastClosed) onExit();
        return;
      }

      // Cmd+] — focus next pane
      if (e.key === "]" && !e.shiftKey) {
        e.preventDefault();
        panes.focusNext();
        return;
      }

      // Cmd+[ — focus prev pane
      if (e.key === "[" && !e.shiftKey) {
        e.preventDefault();
        panes.focusPrev();
        return;
      }
    },
    [onExit, panes],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  return (
    <div className={styles.overlay} data-tour="fullscreen-command">
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <h2 className={styles.modeTitle}>Command Mode</h2>
          <span className={styles.shortcutHint}>
            {panes.leafCount > 1 ? `${panes.leafCount} panes` : "1 pane"} &middot; Esc to exit
          </span>
        </div>
        <div className={styles.headerActions}>
          <button
            className={styles.splitBtn}
            onClick={() => panes.splitPane(panes.activePaneId, "horizontal")}
            title="Split vertical (Cmd+D)"
          >
            Split |
          </button>
          <button
            className={styles.splitBtn}
            onClick={() => panes.splitPane(panes.activePaneId, "vertical")}
            title="Split horizontal (Cmd+Shift+D)"
          >
            Split —
          </button>
          <button className={styles.exitBtn} onClick={onExit} title="Exit (Esc)">
            Exit
          </button>
        </div>
      </div>
      <div className={styles.content}>
        <SplitContainer
          node={panes.root}
          agents={agents}
          activePaneId={panes.activePaneId}
          onFocusPane={panes.setActivePaneId}
          onAgentChange={panes.setAgentId}
          onClosePane={(id) => {
            const lastClosed = panes.closePane(id);
            if (lastClosed) onExit();
          }}
          leafCount={panes.leafCount}
        />
      </div>
    </div>
  );
}
