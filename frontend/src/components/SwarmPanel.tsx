import { useState } from "react";
import type { SwarmStatus, SwarmSettings } from "../types/api";
import styles from "./SwarmPanel.module.css";

interface SwarmPanelProps {
  swarm: SwarmStatus | null;
  loading: boolean;
  starting: boolean;
  stopping: boolean;
  updatingSettings: boolean;
  onStart: () => void;
  onStop: () => void;
  onUpdateSettings: (settings: SwarmSettings) => void;
}

export function SwarmPanel({
  swarm,
  loading,
  starting,
  stopping,
  updatingSettings,
  onStart,
  onStop,
  onUpdateSettings,
}: SwarmPanelProps) {
  const [maxSwarmSize, setMaxSwarmSize] = useState<string>("");
  const [dirty, setDirty] = useState(false);

  // Sync local state when swarm data loads
  const displaySize = dirty ? maxSwarmSize : String(swarm?.max_workers ?? "");

  if (loading) {
    return <p className={styles.empty}>Loading swarm...</p>;
  }

  if (!swarm) {
    return <p className={styles.empty}>Swarm not configured</p>;
  }

  const activeItems = swarm.items.filter((i) => i.status === "assigned");
  const queuedItems = swarm.items.filter((i) => i.status === "queued");

  const handleSave = () => {
    const val = parseInt(displaySize, 10);
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
            {swarm.running ? (
              <span className={styles.runningIndicator} />
            ) : (
              <span className={styles.stoppedIndicator} />
            )}
            {swarm.running ? "Running" : "Stopped"}
          </span>
          <span>
            Agents: <span className={styles.statValue}>{swarm.active_workers} / {swarm.max_workers}</span>
          </span>
          <span>
            <span className={styles.statValue}>{queuedItems.length} queued</span>
          </span>
        </div>
        <div className={styles.controls}>
          {swarm.running ? (
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
        <span className={styles.settingsLabel}>Max Swarm Size:</span>
        <input
          type="number"
          className={styles.sizeInput}
          value={displaySize}
          min={1}
          max={10}
          onChange={(e) => {
            setMaxSwarmSize(e.target.value);
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
            <div key={item.track_id} className={styles.agentRow}>
              <span className={`${styles.statusDot} ${styles.dotRunning}`} />
              <span className={styles.agentName}>{item.agent_id}</span>
              <span className={styles.agentTrack}>{item.track_id}</span>
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
        <p className={styles.empty}>No items in swarm</p>
      )}
    </div>
  );
}
