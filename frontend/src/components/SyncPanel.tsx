import { useState, useCallback } from "react";
import type { SyncStatus } from "../types/api";
import styles from "./SyncPanel.module.css";

interface Props {
  syncStatus: SyncStatus | null;
  loading: boolean;
  pushing: boolean;
  pulling: boolean;
  error: string | null;
  onPush: (remoteBranch: string) => void;
  onPull: (remoteBranch?: string) => void;
  onRefresh: () => void;
  onClearError: () => void;
}

const statusLabels: Record<string, string> = {
  synced: "Synced",
  ahead: "Ahead",
  behind: "Behind",
  diverged: "Diverged",
  unknown: "Unknown",
};

const statusColors: Record<string, string> = {
  synced: "var(--green)",
  ahead: "var(--yellow)",
  behind: "var(--yellow)",
  diverged: "var(--red)",
  unknown: "var(--text-dim)",
};

export function SyncPanel({ syncStatus, loading, pushing, pulling, error, onPush, onPull, onRefresh, onClearError }: Props) {
  const [pushBranch, setPushBranch] = useState("kf/main");
  const [showPushInput, setShowPushInput] = useState(false);

  const handlePush = useCallback(() => {
    onPush(pushBranch);
    setShowPushInput(false);
  }, [onPush, pushBranch]);

  const handlePull = useCallback(() => {
    onPull();
  }, [onPull]);

  const busy = pushing || pulling;

  return (
    <div className={styles.panel}>
      <div className={styles.statusRow}>
        <div className={styles.statusInfo}>
          {loading ? (
            <span className={styles.dim}>Loading sync status...</span>
          ) : syncStatus ? (
            <>
              <span
                className={styles.statusDot}
                style={{ background: statusColors[syncStatus.status] ?? statusColors.unknown }}
              />
              <span className={styles.statusLabel}>
                {statusLabels[syncStatus.status] ?? "Unknown"}
              </span>
              {syncStatus.ahead > 0 && (
                <span className={styles.count}>{syncStatus.ahead} ahead</span>
              )}
              {syncStatus.behind > 0 && (
                <span className={styles.count}>{syncStatus.behind} behind</span>
              )}
              <span className={styles.dim}>branch: {syncStatus.local_branch}</span>
              {syncStatus.remote_url && (
                <span className={styles.dim}>{syncStatus.remote_url}</span>
              )}
            </>
          ) : (
            <span className={styles.dim}>No origin remote configured</span>
          )}
        </div>

        <button className={styles.refreshBtn} onClick={onRefresh} disabled={loading} title="Refresh status">
          &#x21bb;
        </button>
      </div>

      {error && (
        <div className={styles.error}>
          <span>{error}</span>
          <button className={styles.dismissBtn} onClick={onClearError}>&times;</button>
        </div>
      )}

      {syncStatus && (
        <div className={styles.actions}>
          {showPushInput ? (
            <div className={styles.pushForm}>
              <label className={styles.inputLabel}>Remote branch:</label>
              <input
                className={styles.input}
                value={pushBranch}
                onChange={(e) => setPushBranch(e.target.value)}
                placeholder="kf/main"
              />
              <button className={styles.btn} onClick={handlePush} disabled={busy || !pushBranch}>
                {pushing ? "Pushing..." : "Push"}
              </button>
              <button className={styles.btnSecondary} onClick={() => setShowPushInput(false)} disabled={busy}>
                Cancel
              </button>
            </div>
          ) : (
            <>
              <button className={styles.btn} onClick={() => setShowPushInput(true)} disabled={busy}>
                Push to Upstream
              </button>
              <button className={styles.btn} onClick={handlePull} disabled={busy}>
                {pulling ? "Pulling..." : "Pull from Upstream"}
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}
