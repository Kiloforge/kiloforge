import { useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import type { Agent } from "../types/api";
import styles from "./AgentStatusPopover.module.css";

interface Props {
  status: string;
  agents: Agent[];
  onClose: () => void;
}

export function AgentStatusPopover({ status, agents, onClose }: Props) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    }
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [onClose]);

  return (
    <div ref={ref} className={styles.popover}>
      <div className={styles.header}>
        <span className={`${styles.statusDot} ${styles[status] ?? ""}`} />
        {agents.length} {status}
      </div>
      {agents.length === 0 ? (
        <div className={styles.empty}>No agents</div>
      ) : (
        <ul className={styles.list}>
          {agents.map((agent) => (
            <li key={agent.id}>
              <Link
                to={`/agents/${agent.id}`}
                className={styles.agentLink}
                onClick={onClose}
              >
                <span className={styles.agentName}>
                  {agent.name ?? agent.id.slice(0, 8)}
                </span>
                <span className={styles.agentMeta}>
                  {agent.role} &middot; {agent.ref}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
