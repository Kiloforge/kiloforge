import { useState, useEffect, useCallback, useRef } from "react";
import { useCommandPanes } from "../../hooks/useCommandPanes";
import { usePlatform } from "../../hooks/usePlatform";
import type { Agent } from "../../types/api";
import { SplitContainer } from "./SplitContainer";
import { CommandModeHelp } from "./CommandModeHelp";
import styles from "./FullScreenCommand.module.css";

interface Props {
  agents: Agent[];
  onExit: () => void;
}

export function FullScreenCommand({ agents, onExit }: Props) {
  const panes = useCommandPanes();
  const { mod } = usePlatform();
  const [showHelp, setShowHelp] = useState(false);
  const clearFnsRef = useRef(new Map<string, () => void>());

  const registerClear = useCallback((paneId: string, clearFn: () => void) => {
    clearFnsRef.current.set(paneId, clearFn);
    return () => { clearFnsRef.current.delete(paneId); };
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const isMod = e.metaKey || e.ctrlKey;

      // Escape exits full-screen mode (or closes help first)
      if (e.key === "Escape") {
        e.preventDefault();
        if (showHelp) {
          setShowHelp(false);
        } else {
          onExit();
        }
        return;
      }

      if (!isMod) return;

      // Cmd+? — toggle help panel
      if (e.key === "?" || (e.shiftKey && e.key === "/")) {
        e.preventDefault();
        setShowHelp((v) => !v);
        return;
      }

      // Cmd+K — clear active pane messages
      if (e.key === "k" && !e.shiftKey) {
        e.preventDefault();
        clearFnsRef.current.get(panes.activePaneId)?.();
        return;
      }

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
    [onExit, panes, showHelp],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // Browser Fullscreen API: enter on mount, exit on unmount
  useEffect(() => {
    try {
      document.documentElement.requestFullscreen?.();
    } catch {
      // Fullscreen not supported or denied — continue as overlay
    }

    const handleFullscreenChange = () => {
      if (!document.fullscreenElement) {
        onExit();
      }
    };
    document.addEventListener("fullscreenchange", handleFullscreenChange);

    return () => {
      document.removeEventListener("fullscreenchange", handleFullscreenChange);
      try {
        if (document.fullscreenElement) {
          document.exitFullscreen?.();
        }
      } catch {
        // Ignore errors when exiting fullscreen
      }
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

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
            title={`Split vertical (${mod}+D)`}
          >
            Split |
          </button>
          <button
            className={styles.splitBtn}
            onClick={() => panes.splitPane(panes.activePaneId, "vertical")}
            title={`Split horizontal (${mod}+Shift+D)`}
          >
            Split —
          </button>
          <button
            className={styles.helpBtn}
            onClick={() => setShowHelp((v) => !v)}
            title={`Keyboard shortcuts (${mod}+?)`}
          >
            ?
          </button>
          <button className={styles.exitBtn} onClick={onExit} title="Exit (Esc)">
            Exit
          </button>
        </div>
      </div>
      {showHelp && <CommandModeHelp onClose={() => setShowHelp(false)} />}
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
          onRegisterClear={registerClear}
        />
      </div>
    </div>
  );
}
