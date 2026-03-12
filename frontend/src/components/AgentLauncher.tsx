import { useState, useCallback } from "react";
import type { SwarmCapacity } from "../types/api";
import { SKILL_REGISTRY } from "../skills/registry";
import styles from "./AgentLauncher.module.css";

export type AgentRole = string;

interface AgentLauncherProps {
  onLaunch: (role: AgentRole, prompt: string) => void;
  onClose: () => void;
  launching?: boolean;
  projectSlug?: string;
  waitingForCapacity?: boolean;
  waitingCapacity?: SwarmCapacity | null;
  onCancelWaiting?: () => void;
}

export function AgentLauncher({ onLaunch, onClose, launching, projectSlug, waitingForCapacity, waitingCapacity, onCancelWaiting }: AgentLauncherProps) {
  // Only show roles appropriate for the current context.
  const availableRoles = projectSlug
    ? SKILL_REGISTRY
    : SKILL_REGISTRY.filter((r) => !r.requiresProject);
  const [role, setRole] = useState<AgentRole>("interactive");
  const [prompt, setPrompt] = useState("");

  const selectedRole = availableRoles.find((r) => r.role === role) ?? availableRoles[0];

  const handleSubmit = useCallback(() => {
    onLaunch(role, prompt.trim());
  }, [role, prompt, onLaunch]);

  if (waitingForCapacity) {
    const active = waitingCapacity?.active ?? 0;
    const max = waitingCapacity?.max ?? 0;

    return (
      <div className={styles.overlay} onClick={onCancelWaiting}>
        <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
          <div className={styles.waitingOverlay}>
            <div className={styles.waitingPulse} />
            <h3 className={styles.waitingTitle}>Kiloforge at max capacity</h3>
            <p className={styles.waitingUsage}>
              {active}/{max} agents active — increase Max Swarm Size to prevent waiting
            </p>
            <p className={styles.waitingHint}>
              Will auto-retry when a slot opens...
            </p>
            <button
              className={styles.cancelBtn}
              onClick={onCancelWaiting}
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h3 className={styles.title}>New Agent</h3>
        <p className={styles.subtitle}>Choose an agent type and optionally provide a prompt.</p>

        {availableRoles.length > 1 && (
          <div className={styles.roleList}>
            {availableRoles.map((r) => (
              <button
                key={r.role}
                className={`${styles.roleCard} ${role === r.role ? styles.roleCardActive : ""}`}
                onClick={() => setRole(r.role)}
                type="button"
              >
                <span className={styles.roleLabel}>{r.label}</span>
                <span className={styles.roleDesc}>{r.description}</span>
              </button>
            ))}
          </div>
        )}

        <textarea
          className={styles.promptInput}
          placeholder={selectedRole.placeholder}
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          rows={3}
          onKeyDown={(e) => {
            if (e.nativeEvent.isComposing) return;
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
              handleSubmit();
            }
          }}
        />

        <div className={styles.actions}>
          <button className={styles.cancelBtn} onClick={onClose} disabled={launching}>
            Cancel
          </button>
          <button className={styles.startBtn} onClick={handleSubmit} disabled={launching}>
            {launching ? "Starting..." : "Start"}
          </button>
        </div>
      </div>
    </div>
  );
}
