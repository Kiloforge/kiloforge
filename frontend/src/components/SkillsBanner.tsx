import { useSkillsStatus } from "../hooks/useSkillsStatus";
import styles from "./SkillsBanner.module.css";

export function SkillsBanner() {
  const { status, loading, updating, triggerUpdate } = useSkillsStatus();

  if (loading || !status) return null;

  // No skills installed yet.
  if (status.skills.length === 0) {
    return (
      <div className={`${styles.banner} ${styles.warn}`}>
        <span className={styles.message}>
          No skills installed. Install to enable agent spawning and track generation.
        </span>
        <button
          className={styles.btn}
          disabled={updating}
          onClick={() => triggerUpdate(true)}
        >
          {updating ? "Installing..." : "Install Skills"}
        </button>
      </div>
    );
  }

  // Update available.
  if (status.update_available) {
    const hasModified = status.skills.some((s) => s.modified);
    return (
      <div className={`${styles.banner} ${styles.info}`}>
        <span className={styles.message}>
          Skills update available: {status.installed_version || "none"} &rarr;{" "}
          {status.available_version}
          {hasModified && " (some skills have local modifications)"}
        </span>
        <button
          className={styles.btn}
          disabled={updating}
          onClick={() => triggerUpdate(hasModified)}
        >
          {updating ? "Updating..." : "Update"}
        </button>
      </div>
    );
  }

  return null;
}
