import { useEffect, useCallback } from "react";
import { SKILL_REGISTRY } from "../skills/registry";
import { useSkillsStatus } from "../hooks/useSkillsStatus";
import styles from "./SkillsPalette.module.css";

interface SkillsPaletteProps {
  onClose: () => void;
  onSelectSkill: (role: string) => void;
  hasProject?: boolean;
}

export function SkillsPalette({ onClose, onSelectSkill, hasProject = false }: SkillsPaletteProps) {
  const { status } = useSkillsStatus();

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    },
    [onClose],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.panel} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          <h3 className={styles.title}>Skills</h3>
          <button className={styles.closeBtn} onClick={onClose} aria-label="Close">
            &times;
          </button>
        </div>
        <div className={styles.list}>
          {SKILL_REGISTRY.map((entry) => {
            const installed = status?.skills.some((s) => s.name === entry.requiredSkill);
            const modified = status?.skills.find((s) => s.name === entry.requiredSkill)?.modified;
            const needsProject = entry.requiresProject && !hasProject;

            return (
              <button
                key={entry.role}
                className={styles.card}
                onClick={() => onSelectSkill(entry.role)}
              >
                <div className={styles.cardHeader}>
                  <span className={styles.cardLabel}>{entry.label}</span>
                  <span className={styles.badges}>
                    {installed && (
                      <span className={`${styles.badge} ${modified ? styles.badgeWarn : styles.badgeOk}`}>
                        {modified ? "Modified" : "Installed"}
                      </span>
                    )}
                    {!installed && (
                      <span className={`${styles.badge} ${styles.badgeDim}`}>Not installed</span>
                    )}
                    {needsProject && (
                      <span className={`${styles.badge} ${styles.badgeDim}`}>Requires project</span>
                    )}
                  </span>
                </div>
                <span className={styles.cardDesc}>{entry.description}</span>
                <span className={styles.slashCmd}>{entry.slashCommand}</span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
