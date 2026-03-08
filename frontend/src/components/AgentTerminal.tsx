import { useState, useEffect, useRef, useCallback } from "react";
import { useAgentWebSocket } from "../hooks/useAgentWebSocket";
import type { WSMessage, WSConnectionState } from "../hooks/useAgentWebSocket";
import styles from "./AgentTerminal.module.css";

interface Props {
  agentId: string;
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

function formatCode(text: string): React.ReactNode[] {
  // Split on code blocks (```...```) and render them differently
  const parts = text.split(/(```[\s\S]*?```)/g);
  return parts.map((part, i) => {
    if (part.startsWith("```") && part.endsWith("```")) {
      const inner = part.slice(3, -3);
      // Strip optional language tag on first line
      const newlineIdx = inner.indexOf("\n");
      const code = newlineIdx >= 0 ? inner.slice(newlineIdx + 1) : inner;
      return (
        <pre key={i} className={styles.codeBlock}>
          <code>{code}</code>
        </pre>
      );
    }
    // Render inline code with backticks
    const inlineParts = part.split(/(`[^`]+`)/g);
    return (
      <span key={i}>
        {inlineParts.map((ip, j) =>
          ip.startsWith("`") && ip.endsWith("`") ? (
            <code key={j} className={styles.inlineCode}>
              {ip.slice(1, -1)}
            </code>
          ) : (
            <span key={j}>{ip}</span>
          ),
        )}
      </span>
    );
  });
}

function MessageBubble({ msg }: { msg: WSMessage }) {
  if (msg.type === "input") {
    return (
      <div className={`${styles.message} ${styles.userMessage}`}>
        <span className={styles.messageIcon}>you</span>
        <div className={styles.messageContent}>{msg.text}</div>
      </div>
    );
  }

  if (msg.type === "status") {
    return (
      <div className={`${styles.message} ${styles.statusMessage}`}>
        <span className={styles.statusText}>{msg.text}</span>
      </div>
    );
  }

  if (msg.type === "error") {
    return (
      <div className={`${styles.message} ${styles.errorMessage}`}>
        <span className={styles.messageIcon}>err</span>
        <div className={styles.messageContent}>{msg.text}</div>
      </div>
    );
  }

  // output
  return (
    <div className={`${styles.message} ${styles.agentMessage}`}>
      <span className={styles.messageIcon}>kf</span>
      <div className={styles.messageContent}>{formatCode(msg.text)}</div>
    </div>
  );
}

export function AgentTerminal({ agentId, onClose }: Props) {
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

  return (
    <div className={styles.overlay} onClick={handleBackdrop}>
      <div className={styles.modal}>
        <div className={styles.header}>
          <div className={styles.headerLeft}>
            <h3>
              Agent: <span className={styles.agentId}>{agentId}</span>
            </h3>
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
          {messages.map((msg, i) => (
            <MessageBubble key={i} msg={msg} />
          ))}
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
