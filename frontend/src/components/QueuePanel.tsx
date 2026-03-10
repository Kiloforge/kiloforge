import { useState } from "react";
import type { QueueStatus, QueueSettings } from "../types/api";
import styles from "./QueuePanel.module.css";

interface QueuePanelProps {
  queue: QueueStatus | null;
  loading: boolean;
  starting: boolean;
  stopping: boolean;
  updatingSettings: boolean;
  onStart: () => void;
  onStop: () => void;
  onUpdateSettings: (settings: QueueSettings) => void;
}

export function QueuePanel({
  queue,
  loading,
  starting,
  stopping,
  updatingSettings,
  onStart,
  onStop,
  onUpdateSettings,
}: QueuePanelProps) {
  const [maxWorkers, setMaxWorkers] = useState<string>("");
  const [dirty, setDirty] = useState(false);

  // Sync local state when queue data loads
  const displayWorkers = dirty ? maxWorkers : String(queue?.max_workers ?? "");

  if (loading) {
    return <p className={styles.empty}>Loading queue...</p>;
  }

  if (!queue) {
    return <p className={styles.empty}>Queue not configured</p>;
  }

  const activeItems = queue.items.filter((i) => i.status === "assigned");
  const queuedItems = queue.items.filter((i) => i.status === "queued");

  const handleSave = () => {
    const val = parseInt(displayWorkers, 10);
    if (val > 0 && val <= 10) {
      onUpdateSettings({ max_workers: val });
      setDirty(false);
    }
  };

  return (
    <div>
      <div className={styles.header}>
        <div className={styles.stats}>
          <span>
            {queue.running ? (
              <span className={styles.runningIndicator} />
            ) : (
              <span className={styles.stoppedIndicator} />
            )}
            {queue.running ? "Running" : "Stopped"}
          </span>
          <span>
            Workers: <span className={styles.statValue}>{queue.active_workers} / {queue.max_workers}</span>
          </span>
          <span>
            <span className={styles.statValue}>{queuedItems.length} queued</span>
          </span>
        </div>
        <div className={styles.controls}>
          {queue.running ? (
            <button
              className={`${styles.btn} ${styles.btnStop}`}
              onClick={onStop}
              disabled={stopping}
            >
              {stopping ? "Stopping..." : "Stop"}
            </button>
          ) : (
            <button
              className={`${styles.btn} ${styles.btnStart}`}
              onClick={onStart}
              disabled={starting}
            >
              {starting ? "Starting..." : "Start"}
            </button>
          )}
        </div>
      </div>

      <div className={styles.settingsRow}>
        <span className={styles.settingsLabel}>Max workers:</span>
        <input
          type="number"
          className={styles.workersInput}
          value={displayWorkers}
          min={1}
          max={10}
          onChange={(e) => {
            setMaxWorkers(e.target.value);
            setDirty(true);
          }}
        />
        {dirty && (
          <button
            className={styles.saveBtn}
            onClick={handleSave}
            disabled={updatingSettings}
          >
            {updatingSettings ? "Saving..." : "Save"}
          </button>
        )}
      </div>

      {activeItems.length > 0 && (
        <div className={styles.section}>
          <div className={styles.sectionTitle}>Active</div>
          {activeItems.map((item) => (
            <div key={item.track_id} className={styles.workerRow}>
              <span className={`${styles.statusDot} ${styles.dotRunning}`} />
              <span className={styles.workerAgent}>{item.agent_id}</span>
              <span className={styles.workerTrack}>{item.track_id}</span>
            </div>
          ))}
        </div>
      )}

      {queuedItems.length > 0 && (
        <div className={styles.section}>
          <div className={styles.sectionTitle}>Queued</div>
          {queuedItems.map((item, idx) => (
            <div key={item.track_id} className={styles.queuedRow}>
              <span className={`${styles.statusDot} ${styles.dotQueued}`} />
              <span className={styles.queuedIndex}>{idx + 1}.</span>
              <span className={styles.queuedTrack}>{item.track_id}</span>
            </div>
          ))}
        </div>
      )}

      {activeItems.length === 0 && queuedItems.length === 0 && (
        <p className={styles.empty}>No items in queue</p>
      )}
    </div>
  );
}
