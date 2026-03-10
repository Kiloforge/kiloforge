import { usePlatform } from "../../hooks/usePlatform";
import type { ShortcutEntry } from "../../hooks/useKeyboardShortcuts";
import styles from "./FullScreenCommand.module.css";

function getCommandModeShortcuts(isMac: boolean): ShortcutEntry[] {
  const mod = isMac ? "⌘" : "Ctrl+";
  const shift = isMac ? "⇧" : "Shift+";
  return [
    { keys: `${mod}D`, description: "Split pane vertically" },
    { keys: `${mod}${shift}D`, description: "Split pane horizontally" },
    { keys: `${mod}W`, description: "Close active pane" },
    { keys: `${mod}]`, description: "Focus next pane" },
    { keys: `${mod}[`, description: "Focus previous pane" },
    { keys: `${mod}K`, description: "Clear pane messages" },
    { keys: `${mod}?`, description: "Toggle this help panel" },
    { keys: "Esc", description: "Exit command mode" },
  ];
}

interface Props {
  onClose: () => void;
}

export function CommandModeHelp({ onClose }: Props) {
  const { isMac } = usePlatform();
  const shortcuts = getCommandModeShortcuts(isMac);

  return (
    <div className={styles.helpPanel} data-testid="command-mode-help">
      <div className={styles.helpHeader}>
        <span className={styles.helpTitle}>Keyboard Shortcuts</span>
        <button className={styles.helpCloseBtn} onClick={onClose} title="Close help">
          &times;
        </button>
      </div>
      <div className={styles.helpGrid}>
        {shortcuts.map((entry) => (
          <div key={entry.keys} className={styles.helpRow}>
            <kbd className={styles.helpKeys}>{entry.keys}</kbd>
            <span className={styles.helpDesc}>{entry.description}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
