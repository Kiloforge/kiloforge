import { useState, useEffect, useRef, useCallback } from "react";
import { useAgentWebSocket } from "../hooks/useAgentWebSocket";
import type { WSConnectionState } from "../hooks/useAgentWebSocket";
import { MessageDispatch } from "./terminal";
import styles from "./AgentTerminal.module.css";

interface Props {
  agentId: string;
  name?: string;
  role?: string;
  onClose: () => void;
}

function ConnectionDot({ status }: { status: WSConnectionState }) {
  const cls =
    status === "connected"
      ? styles.dotConnected
      : status === "reconnecting" || status === "connecting"
        ? styles.dotReconnecting
        : styles.dotDisconnected;
  const label =
    status === "connected"
      ? "Connected"
      : status === "connecting"
        ? "Connecting..."
        : status === "reconnecting"
          ? "Reconnecting..."
          : "Disconnected";
  return (
    <span className={styles.connectionStatus}>
      <span className={`${styles.dot} ${cls}`} />
      <span className={styles.connectionLabel}>{label}</span>
    </span>
  );
}

export function AgentTerminal({ agentId, name, role, onClose }: Props) {
  const { messages, sendMessage, status, agentStatus } = useAgentWebSocket(agentId);
  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Auto-scroll on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const handleSend = useCallback(() => {
    const text = input.trim();
    if (!text) return;
    sendMessage(text);
    setInput("");
    inputRef.current?.focus();
  }, [input, sendMessage]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend],
  );

  const handleBackdrop = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === e.currentTarget) onClose();
    },
    [onClose],
  );

  const isTerminal = agentStatus === "completed" || agentStatus === "failed";
  const canSend = status === "connected" && !isTerminal;

  // Compute turn numbers for turn_start messages
  let turnCounter = 0;

  return (
    <div className={styles.overlay} onClick={handleBackdrop}>
      <div className={styles.modal}>
        <div className={styles.header}>
          <div className={styles.headerLeft}>
            <h3>
              {name || agentId.slice(0, 8)}
              {role && <span className={`${styles.roleBadge} ${styles[role] ?? ""}`}>{role}</span>}
            </h3>
            <span className={styles.agentIdSecondary}>{agentId.slice(0, 8)}</span>
            <ConnectionDot status={status} />
          </div>
          <button className={styles.closeBtn} onClick={onClose}>
            &times;
          </button>
        </div>

        <div className={styles.messages}>
          {messages.length === 0 && status === "connecting" && (
            <p className={styles.emptyState}>Connecting to agent...</p>
          )}
          {messages.length === 0 && status === "connected" && (
            <p className={styles.emptyState}>Waiting for agent output...</p>
          )}
          {messages.map((msg, i) => {
            if (msg.type === "turn_start") turnCounter++;
            return <MessageDispatch key={i} msg={msg} turnNumber={turnCounter} />;
          })}
          <div ref={messagesEndRef} />
        </div>

        <div className={styles.inputArea}>
          <textarea
            ref={inputRef}
            className={styles.inputField}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={canSend ? "Type a message... (Enter to send)" : isTerminal ? "Agent has exited" : "Connecting..."}
            disabled={!canSend}
            rows={1}
          />
          <button className={styles.sendBtn} onClick={handleSend} disabled={!canSend || !input.trim()}>
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
