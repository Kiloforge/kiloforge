import { useState, useCallback } from "react";
import styles from "./AgentLauncher.module.css";

export type AgentRole = "interactive" | "architect" | "product-advisor";

interface RoleOption {
  value: AgentRole;
  label: string;
  description: string;
  placeholder: string;
}

const ROLES: RoleOption[] = [
  {
    value: "architect",
    label: "Architect",
    description: "Research codebase and generate implementation tracks",
    placeholder: "Describe the features or changes you want to plan...",
  },
  {
    value: "product-advisor",
    label: "Product Advisor",
    description: "Product design, branding, and competitive analysis",
    placeholder: "Describe what you need product guidance on...",
  },
];

interface AgentLauncherProps {
  onLaunch: (role: AgentRole, prompt: string) => void;
  onClose: () => void;
  launching?: boolean;
  projectSlug?: string;
}

export function AgentLauncher({ onLaunch, onClose, launching, projectSlug }: AgentLauncherProps) {
  const [role, setRole] = useState<AgentRole>(projectSlug ? "architect" : "architect");
  const [prompt, setPrompt] = useState("");

  const selectedRole = ROLES.find((r) => r.value === role) ?? ROLES[0];

  const handleSubmit = useCallback(() => {
    onLaunch(role, prompt.trim());
  }, [role, prompt, onLaunch]);

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h3 className={styles.title}>New Agent</h3>
        <p className={styles.subtitle}>Choose an agent type and optionally provide a prompt.</p>

        <div className={styles.roleList}>
          {ROLES.map((r) => (
            <button
              key={r.value}
              className={`${styles.roleCard} ${role === r.value ? styles.roleCardActive : ""}`}
              onClick={() => setRole(r.value)}
              type="button"
            >
              <span className={styles.roleLabel}>{r.label}</span>
              <span className={styles.roleDesc}>{r.description}</span>
            </button>
          ))}
        </div>

        <textarea
          className={styles.promptInput}
          placeholder={selectedRole.placeholder}
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          rows={3}
          onKeyDown={(e) => {
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
