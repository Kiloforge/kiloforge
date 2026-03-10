import { useState, useCallback } from "react";
import type { Agent, SwarmCapacity } from "../types/api";
import type { AgentRole } from "./AgentLauncher";
import { AgentGrid } from "./AgentGrid";
import { HelpTooltip } from "./HelpTooltip";
import styles from "./AdvisorHub.module.css";
import appStyles from "../App.module.css";

interface AdvisorType {
  role: AgentRole;
  label: string;
  description: string;
  placeholder: string;
}

const ADVISORS: AdvisorType[] = [
  {
    role: "advisor-product",
    label: "Product Advisor",
    description: "Product strategy, branding, competitive analysis, and feature prioritization. Produces actionable reports for the architect.",
    placeholder: "What do you need product guidance on? (e.g., competitive analysis, branding strategy, feature prioritization)",
  },
  {
    role: "advisor-reliability",
    label: "Reliability Advisor",
    description: "Testing coverage, linting strictness, type safety, CI gates, and dependency security audits. Generates improvement tracks.",
    placeholder: "What should be audited? (e.g., test coverage gaps, CI gate strictness, dependency security)",
  },
];

function isAdvisorRole(role: string): boolean {
  return role.startsWith("advisor-");
}

interface AdvisorHubProps {
  agents: Agent[];
  onLaunch: (role: AgentRole, prompt: string) => void;
  launching?: boolean;
  onViewLog: (agentId: string) => void;
  onAttach?: (agentId: string) => void;
  waitingForCapacity?: boolean;
  waitingCapacity?: SwarmCapacity | null;
  onCancelWaiting?: () => void;
}

export { isAdvisorRole };

export function AdvisorHub({ agents, onLaunch, launching, onViewLog, onAttach, waitingForCapacity, waitingCapacity, onCancelWaiting }: AdvisorHubProps) {
  const [activeAdvisor, setActiveAdvisor] = useState<AdvisorType | null>(null);
  const [prompt, setPrompt] = useState("");

  const advisorAgents = agents.filter((a) => isAdvisorRole(a.role));

  const handleLaunch = useCallback(() => {
    if (!activeAdvisor) return;
    onLaunch(activeAdvisor.role, prompt.trim());
    setPrompt("");
    setActiveAdvisor(null);
  }, [activeAdvisor, prompt, onLaunch]);

  const handleClose = useCallback(() => {
    setActiveAdvisor(null);
    setPrompt("");
  }, []);

  return (
    <section className={appStyles.panel}>
      <div className={styles.header}>
        <h2 className={appStyles.panelTitle}>
          Advisor Hub
          <HelpTooltip term="Advisor Hub" definition="Specialized AI advisors that analyze your project and produce actionable reports. Product advisors guide strategy; reliability advisors audit code quality." />
        </h2>
      </div>

      <div className={styles.advisorGrid}>
        {ADVISORS.map((advisor) => (
          <button
            key={advisor.role}
            className={styles.advisorCard}
            onClick={() => setActiveAdvisor(advisor)}
            type="button"
          >
            <span className={styles.advisorLabel}>{advisor.label}</span>
            <span className={styles.advisorDesc}>{advisor.description}</span>
            <span className={styles.launchHint}>Click to launch</span>
          </button>
        ))}
      </div>

      {advisorAgents.length > 0 && (
        <div className={styles.recentSection}>
          <h3 className={styles.recentTitle}>Recent Advisor Agents</h3>
          <AgentGrid agents={advisorAgents} onViewLog={onViewLog} onAttach={onAttach} />
        </div>
      )}

      {activeAdvisor && !waitingForCapacity && (
        <div className={styles.overlay} onClick={handleClose}>
          <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
            <h3 className={styles.dialogTitle}>{activeAdvisor.label}</h3>
            <p className={styles.dialogDesc}>{activeAdvisor.description}</p>

            <textarea
              className={styles.promptInput}
              placeholder={activeAdvisor.placeholder}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              rows={3}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  handleLaunch();
                }
              }}
            />

            <div className={styles.actions}>
              <button className={styles.cancelBtn} onClick={handleClose} disabled={launching}>
                Cancel
              </button>
              <button className={styles.startBtn} onClick={handleLaunch} disabled={launching}>
                {launching ? "Starting..." : "Start Advisor"}
              </button>
            </div>
          </div>
        </div>
      )}

      {waitingForCapacity && (
        <div className={styles.overlay} onClick={onCancelWaiting}>
          <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
            <div className={styles.waitingOverlay}>
              <div className={styles.waitingPulse} />
              <h3 className={styles.dialogTitle}>Kiloforge at max capacity</h3>
              <p className={styles.waitingUsage}>
                {waitingCapacity?.active ?? 0}/{waitingCapacity?.max ?? 0} agents active
              </p>
              <p className={styles.waitingHint}>Will auto-retry when a slot opens...</p>
              <button className={styles.cancelBtn} onClick={onCancelWaiting}>
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
