import { useState, useEffect } from "react";
import type { Project, Agent, Track } from "../types/api";
import styles from "./GettingStartedChecklist.module.css";

const STORAGE_KEY = "kf_getting_started_dismissed";

interface ChecklistItem {
  label: string;
  hint: string;
  done: boolean;
}

interface GettingStartedChecklistProps {
  projects: Project[];
  agents: Agent[];
  tracks: Track[];
}

export function GettingStartedChecklist({ projects, agents, tracks }: GettingStartedChecklistProps) {
  const [dismissed, setDismissed] = useState(
    () => localStorage.getItem(STORAGE_KEY) === "1",
  );

  const items: ChecklistItem[] = [
    { label: "Kiloforge is running", hint: "You're here!", done: true },
    { label: "Add a project", hint: "kf add <remote>", done: projects.length > 0 },
    { label: "Generate tracks", hint: "kf tracks generate", done: tracks.length > 0 },
    { label: "Spawn your first agent", hint: "kf agent spawn", done: agents.length > 0 },
  ];

  const allDone = items.every((i) => i.done);

  useEffect(() => {
    if (allDone && !dismissed) {
      localStorage.setItem(STORAGE_KEY, "1");
      setDismissed(true);
    }
  }, [allDone, dismissed]);

  if (dismissed || projects.length >= 2) return null;

  const completedCount = items.filter((i) => i.done).length;

  const handleDismiss = () => {
    localStorage.setItem(STORAGE_KEY, "1");
    setDismissed(true);
  };

  return (
    <div className={styles.checklist}>
      <div className={styles.header}>
        <h3 className={styles.title}>Getting Started</h3>
        <span className={styles.progress}>{completedCount} / {items.length}</span>
        <button className={styles.dismiss} onClick={handleDismiss} title="Dismiss checklist">
          &times;
        </button>
      </div>
      <ul className={styles.items}>
        {items.map((item) => (
          <li key={item.label} className={styles.item} data-done={String(item.done)}>
            <span className={styles.check}>{item.done ? "\u2713" : "\u25CB"}</span>
            <span className={styles.label}>{item.label}</span>
            {!item.done && <code className={styles.hint}>{item.hint}</code>}
          </li>
        ))}
      </ul>
    </div>
  );
}
