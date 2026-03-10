import { useState, useCallback } from "react";
import type { AgentRegister as AgentRegisterType, AgentIdentity } from "../types/api";
import styles from "./AgentRegister.module.css";

interface Props {
  register: AgentRegisterType;
}

export function AgentRegister({ register }: Props) {
  const { created_by, claimed_by } = register;

  if (!created_by && !claimed_by) return null;

  return (
    <div className={styles.register}>
      {created_by && <IdentityRow label="Created by" identity={created_by} />}
      {claimed_by && <IdentityRow label="Claimed by" identity={claimed_by} />}
    </div>
  );
}

function IdentityRow({ label, identity }: { label: string; identity: AgentIdentity }) {
  return (
    <div className={styles.row}>
      <span className={styles.label}>{label}</span>
      <div className={styles.details}>
        <div className={styles.primaryLine}>
          {identity.role && <span className={styles.role}>{identity.role}</span>}
          {identity.agent_id && (
            <span className={styles.agentId} title={identity.agent_id}>
              {identity.agent_id.slice(0, 12)}
            </span>
          )}
          {identity.timestamp && (
            <span className={styles.timestamp}>
              {new Date(identity.timestamp).toLocaleString()}
            </span>
          )}
        </div>
        {identity.session_id && (
          <div className={styles.sessionLine}>
            <span className={styles.sessionLabel}>session:</span>
            <code className={styles.sessionId}>{identity.session_id}</code>
            <CopyButton text={identity.session_id} />
          </div>
        )}
        {identity.worktree && (
          <div className={styles.metaLine}>
            <span className={styles.metaKey}>worktree:</span>
            <span className={styles.metaValue}>{identity.worktree}</span>
          </div>
        )}
        {identity.branch && (
          <div className={styles.metaLine}>
            <span className={styles.metaKey}>branch:</span>
            <span className={styles.metaValue}>{identity.branch}</span>
          </div>
        )}
        {identity.model && (
          <div className={styles.metaLine}>
            <span className={styles.metaKey}>model:</span>
            <span className={styles.metaValue}>{identity.model}</span>
          </div>
        )}
      </div>
    </div>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  }, [text]);

  return (
    <button className={styles.copyBtn} onClick={handleCopy} title="Copy to clipboard">
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}
