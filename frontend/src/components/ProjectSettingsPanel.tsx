import { useState, useCallback, useEffect } from "react";
import type { ProjectSettings } from "../types/api";
import styles from "./ProjectSettingsPanel.module.css";

interface Props {
  settings: ProjectSettings | null;
  loading: boolean;
  updating: boolean;
  onUpdate: (req: Partial<ProjectSettings>) => Promise<boolean>;
}

const SETTINGS_META: Record<
  keyof ProjectSettings,
  { label: string; description: string }
> = {
  primary_branch: {
    label: "Primary Branch",
    description: "The branch agents read track state from",
  },
  enforce_dep_ordering: {
    label: "Enforce Dependency Ordering",
    description:
      "AI Agent Swarm skips tracks with unmet dependencies and tries the next one",
  },
};

export function ProjectSettingsPanel({ settings, loading, updating, onUpdate }: Props) {
  const [branchValue, setBranchValue] = useState("");
  const [branchDirty, setBranchDirty] = useState(false);

  useEffect(() => {
    if (settings && !branchDirty) {
      setBranchValue(settings.primary_branch);
    }
  }, [settings, branchDirty]);

  const handleToggle = useCallback(
    (checked: boolean) => {
      onUpdate({ enforce_dep_ordering: checked });
    },
    [onUpdate],
  );

  const handleSaveBranch = useCallback(() => {
    const trimmed = branchValue.trim();
    if (!trimmed || trimmed === settings?.primary_branch) return;
    onUpdate({ primary_branch: trimmed }).then((ok) => {
      if (ok) setBranchDirty(false);
    });
  }, [branchValue, settings, onUpdate]);

  if (loading) {
    return <p className={styles.loading}>Loading settings...</p>;
  }

  if (!settings) {
    return <p className={styles.error}>Failed to load project settings.</p>;
  }

  return (
    <div className={styles.panel}>
      {/* Primary Branch */}
      <div className={styles.settingRow}>
        <span className={styles.settingLabel}>
          {SETTINGS_META.primary_branch.label}
        </span>
        <span className={styles.settingDescription}>
          {SETTINGS_META.primary_branch.description}
        </span>
        <div className={styles.inputRow}>
          <input
            className={styles.input}
            value={branchValue}
            onChange={(e) => {
              setBranchValue(e.target.value);
              setBranchDirty(true);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleSaveBranch();
            }}
            placeholder="main"
          />
          <button
            className={styles.saveBtn}
            onClick={handleSaveBranch}
            disabled={
              updating ||
              !branchDirty ||
              !branchValue.trim() ||
              branchValue.trim() === settings.primary_branch
            }
          >
            {updating ? "Saving..." : "Save"}
          </button>
        </div>
      </div>

      <hr className={styles.divider} />

      {/* Enforce Dependency Ordering */}
      <div className={styles.settingRow}>
        <div className={styles.settingHeader}>
          <div>
            <span className={styles.settingLabel}>
              {SETTINGS_META.enforce_dep_ordering.label}
            </span>
          </div>
          <label className={styles.toggle}>
            <input
              type="checkbox"
              checked={settings.enforce_dep_ordering}
              onChange={(e) => handleToggle(e.target.checked)}
              disabled={updating}
            />
            <span className={styles.toggleTrack} />
            <span className={styles.toggleThumb} />
          </label>
        </div>
        <span className={styles.settingDescription}>
          {SETTINGS_META.enforce_dep_ordering.description}
        </span>
      </div>
    </div>
  );
}
