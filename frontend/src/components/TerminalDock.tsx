import type { WindowEntry } from "../hooks/useWindowManager";
import styles from "./TerminalDock.module.css";

interface Props {
  windows: WindowEntry[];
  onRestore: (agentId: string) => void;
  onClose: (agentId: string) => void;
}

const roleColors: Record<string, string> = {
  developer: styles.roleDeveloper,
  reviewer: styles.roleReviewer,
  interactive: styles.roleInteractive,
};

export function TerminalDock({ windows, onRestore, onClose }: Props) {
  if (windows.length === 0) return null;

  return (
    <div className={styles.dock}>
      {windows.map((entry) => (
        <button
          key={entry.agentId}
          className={styles.pill}
          onClick={() => onRestore(entry.agentId)}
          title={entry.name || entry.agentId}
        >
          <span className={`${styles.roleDot} ${roleColors[entry.role ?? ""] ?? ""}`} />
          <span className={styles.pillName}>
            {entry.name || entry.agentId.slice(0, 8)}
          </span>
          {entry.unreadCount > 0 && (
            <span className={styles.badge}>
              {entry.unreadCount > 99 ? "99+" : entry.unreadCount}
            </span>
          )}
          <span
            className={styles.pillClose}
            onClick={(e) => {
              e.stopPropagation();
              onClose(entry.agentId);
            }}
          >
            &times;
          </span>
        </button>
      ))}
    </div>
  );
}
