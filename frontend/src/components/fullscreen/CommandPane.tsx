import { useState, useEffect, useRef, useCallback } from "react";
import { useAgentWebSocket } from "../../hooks/useAgentWebSocket";
import type { WSConnectionState } from "../../hooks/useAgentWebSocket";
import type { Agent } from "../../types/api";
import { MessageDispatch } from "../terminal";
import styles from "./FullScreenCommand.module.css";

interface Props {
  paneId: string;
  agentId: string | null;
  agents: Agent[];
  isFocused: boolean;
  onFocus: () => void;
  onAgentChange: (agentId: string | null) => void;
  onClose: () => void;
  showCloseBtn: boolean;
}

function ConnectionDot({ status }: { status: WSConnectionState }) {
  const cls =
    status === "connected"
      ? styles.dotConnected
      : status === "reconnecting" || status === "connecting"
        ? styles.dotReconnecting
        : styles.dotDisconnected;
  return <span className={`${styles.connectionDot} ${cls}`} />;
}

export function CommandPane({
  paneId,
  agentId,
  agents,
  isFocused,
  onFocus,
  onAgentChange,
  onClose,
  showCloseBtn,
}: Props) {
  const { messages, sendMessage, status, agentStatus } = useAgentWebSocket(agentId);
  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Auto-scroll on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Focus input when pane becomes focused
  useEffect(() => {
    if (isFocused) {
      inputRef.current?.focus();
    }
  }, [isFocused]);

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

  const isTerminal = agentStatus === "completed" || agentStatus === "failed";
  const canSend = agentId !== null && status === "connected" && !isTerminal;

  const activeAgents = agents.filter(
    (a) => a.status === "running" || a.status === "interactive" || a.status === "suspended",
  );

  let turnCounter = 0;

  return (
    <div
      className={`${styles.pane} ${isFocused ? styles.paneFocused : ""}`}
      onClick={onFocus}
      data-pane-id={paneId}
    >
      <div className={styles.paneHeader}>
        <div className={styles.paneHeaderLeft}>
          <select
            className={styles.agentSelect}
            value={agentId ?? ""}
            onChange={(e) => onAgentChange(e.target.value || null)}
          >
            <option value="">Select agent...</option>
            {activeAgents.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name || a.id.slice(0, 8)} ({a.role})
              </option>
            ))}
          </select>
          {agentId && <ConnectionDot status={status} />}
          {agentId && agents.find((a) => a.id === agentId)?.role && (
            <span className={`${styles.roleBadge} ${styles[agents.find((a) => a.id === agentId)!.role] ?? ""}`}>
              {agents.find((a) => a.id === agentId)!.role}
            </span>
          )}
        </div>
        {showCloseBtn && (
          <button className={styles.paneCloseBtn} onClick={onClose} title="Close pane">
            &times;
          </button>
        )}
      </div>

      <div className={styles.messages}>
        {!agentId && (
          <p className={styles.emptyState}>
            Select an agent to connect
            <br />
            <span className={styles.emptyHint}>Use the dropdown above to pick a running agent</span>
          </p>
        )}
        {agentId && messages.length === 0 && status === "connecting" && (
          <p className={styles.emptyState}>Connecting to agent...</p>
        )}
        {agentId && messages.length === 0 && status === "connected" && (
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
          placeholder={
            !agentId
              ? "Select an agent first..."
              : canSend
                ? "Type a message... (Enter to send)"
                : isTerminal
                  ? "Agent has exited"
                  : "Connecting..."
          }
          disabled={!canSend}
          rows={1}
        />
        <button className={styles.sendBtn} onClick={handleSend} disabled={!canSend || !input.trim()}>
          Send
        </button>
      </div>
    </div>
  );
}
