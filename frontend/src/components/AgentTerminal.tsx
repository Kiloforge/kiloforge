import { useState, useEffect, useRef, useCallback } from "react";
import { useAgentWebSocket } from "../hooks/useAgentWebSocket";
import type { WSConnectionState } from "../hooks/useAgentWebSocket";
import { useFloatingWindow, detectEdge, cursorForEdge } from "../hooks/useFloatingWindow";
import { MessageDispatch } from "./terminal";
import styles from "./AgentTerminal.module.css";

interface Props {
  agentId: string;
  name?: string;
  role?: string;
  initialX?: number;
  initialY?: number;
  minimized?: boolean;
  onClose: () => void;
  onFocus?: () => void;
  onMinimize?: () => void;
  onActivity?: () => void;
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

export function AgentTerminal({ agentId, name, role, initialX, initialY, minimized, onClose, onFocus, onMinimize, onActivity }: Props) {
  const { messages, sendMessage, status, agentStatus } = useAgentWebSocket(agentId);
  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  const fw = useFloatingWindow({ defaultWidth: 720, defaultHeight: 500, minWidth: 400, minHeight: 300, initialX, initialY });

  const prevMsgCountRef = useRef(0);

  // Auto-scroll on new messages (only when visible)
  useEffect(() => {
    if (!minimized) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, minimized]);

  // Notify activity when new messages arrive while minimized
  useEffect(() => {
    if (minimized && messages.length > prevMsgCountRef.current) {
      onActivity?.();
    }
    prevMsgCountRef.current = messages.length;
  }, [messages.length, minimized, onActivity]);

  // Focus input on mount and reset position
  useEffect(() => {
    inputRef.current?.focus();
    fw.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
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

  const handlePanelPointerDown = useCallback(
    (e: React.PointerEvent) => {
      fw.bringToFront();
      onFocus?.();
      if (!panelRef.current) return;
      const rect = panelRef.current.getBoundingClientRect();
      const edge = detectEdge(e.clientX, e.clientY, rect);
      if (edge) {
        fw.onResizeStart(e, edge);
      }
    },
    [fw, onFocus],
  );

  const handlePanelPointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (fw.isResizing) {
        fw.onResizeMove(e);
        return;
      }
      if (fw.isDragging) {
        fw.onDragMove(e);
        return;
      }
      // Update hover cursor for resize edges
      if (!panelRef.current) return;
      const rect = panelRef.current.getBoundingClientRect();
      const edge = detectEdge(e.clientX, e.clientY, rect);
      fw.setHoverEdge(edge);
    },
    [fw],
  );

  const handlePanelPointerUp = useCallback(() => {
    if (fw.isResizing) fw.onResizeEnd();
    if (fw.isDragging) fw.onDragEnd();
  }, [fw]);

  const isTerminal = agentStatus === "completed" || agentStatus === "failed";
  const canSend = status === "connected" && !isTerminal;

  // Compute turn numbers for turn_start messages
  let turnCounter = 0;

  const edgeCursor = cursorForEdge(fw.isResizing ? fw.resizeEdge : fw.hoverEdge);

  return (
    <div
      ref={panelRef}
      className={styles.panel}
      style={{
        position: "fixed",
        left: fw.x,
        top: fw.y,
        width: fw.width,
        height: fw.height,
        zIndex: fw.zIndex,
        cursor: edgeCursor || undefined,
        display: minimized ? "none" : undefined,
      }}
      onPointerDown={handlePanelPointerDown}
      onPointerMove={handlePanelPointerMove}
      onPointerUp={handlePanelPointerUp}
    >
      <div
        className={styles.header}
        style={{ cursor: fw.isDragging ? "grabbing" : "grab" }}
        onPointerDown={fw.onDragStart}
        onPointerMove={fw.isDragging ? fw.onDragMove : undefined}
        onPointerUp={fw.isDragging ? fw.onDragEnd : undefined}
      >
        <div className={styles.headerLeft}>
          <h3>
            {name || agentId.slice(0, 8)}
            {role && <span className={`${styles.roleBadge} ${styles[role] ?? ""}`}>{role}</span>}
          </h3>
          <span className={styles.agentIdSecondary}>{agentId.slice(0, 8)}</span>
          <ConnectionDot status={status} />
        </div>
        <div className={styles.headerActions}>
          {onMinimize && (
            <button className={styles.minimizeBtn} onClick={onMinimize} title="Minimize">
              &#x2013;
            </button>
          )}
          <button className={styles.closeBtn} onClick={onClose}>
            &times;
          </button>
        </div>
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
  );
}
