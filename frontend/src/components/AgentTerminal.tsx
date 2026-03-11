import { useState, useEffect, useRef, useCallback } from "react";
import { useAgentWebSocket } from "../hooks/useAgentWebSocket";
import type { WSConnectionState } from "../hooks/useAgentWebSocket";
import { useFloatingWindow, detectEdge, cursorForEdge } from "../hooks/useFloatingWindow";
import { MessageDispatch, MessageErrorBoundary } from "./terminal";
import { AgentDiffPanel } from "./diff/AgentDiffPanel";
import { ThinkingIndicator } from "./ThinkingIndicator";
import styles from "./AgentTerminal.module.css";

const TERMINAL_STATUSES = new Set(["completed", "failed", "stopped", "force-killed", "resume-failed", "replaced", "suspended"]);

interface Props {
  agentId: string;
  name?: string;
  role?: string;
  slug?: string;
  branch?: string;
  initialX?: number;
  initialY?: number;
  minimized?: boolean;
  onClose: () => void;
  onFocus?: () => void;
  onMinimize?: () => void;
  onActivity?: () => void;
  onNotification?: (type: "waiting" | "done" | "unread" | null) => void;
  registerControls?: (agentId: string, controls: { setRect: (x: number, y: number, w: number, h: number) => void; getRect: () => { x: number; y: number; width: number; height: number; zIndex: number }; bringToFront: () => void }) => void;
  unregisterControls?: (agentId: string) => void;
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

export function AgentTerminal({ agentId, name, role, slug, branch, initialX, initialY, minimized, onClose, onFocus, onMinimize, onActivity, onNotification, registerControls, unregisterControls }: Props) {
  const { messages, sendMessage, sendInterrupt, status, agentStatus, turnActive } = useAgentWebSocket(agentId);
  const [queueDepth, setQueueDepth] = useState(0);
  const [input, setInput] = useState("");
  const [viewMode, setViewMode] = useState<"chat" | "diff">("chat");
  const hasDiff = !!(slug && branch);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  const fw = useFloatingWindow({ defaultWidth: 720, defaultHeight: 500, minWidth: 400, minHeight: 300, initialX, initialY });

  // Register floating window controls with the window manager
  useEffect(() => {
    if (registerControls) {
      registerControls(agentId, {
        setRect: fw.setRect,
        getRect: fw.getRect,
        bringToFront: fw.bringToFront,
      });
    }
    return () => {
      if (unregisterControls) unregisterControls(agentId);
    };
  }, [agentId, fw.setRect, fw.getRect, fw.bringToFront, registerControls, unregisterControls]);

  const prevMsgCountRef = useRef(0);

  // Auto-scroll on new messages (only when visible)
  useEffect(() => {
    if (!minimized) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, minimized]);

  // Reset queue depth when turn ends (backend will have delivered queued messages)
  useEffect(() => {
    if (!turnActive) setQueueDepth(0);
  }, [turnActive]);

  // Set notification type based on turn state and agent status when minimized
  useEffect(() => {
    if (!minimized || !onNotification) return;
    if (agentStatus && TERMINAL_STATUSES.has(agentStatus)) {
      onNotification("done");
    } else if (!turnActive) {
      onNotification("waiting");
    } else {
      onNotification(null);
    }
  }, [minimized, turnActive, agentStatus, onNotification]);

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
    if (turnActive) setQueueDepth((d) => d + 1);
    setInput("");
    inputRef.current?.focus();
  }, [input, sendMessage, turnActive]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
      if (e.key === "Escape" && turnActive) {
        e.preventDefault();
        sendInterrupt();
      }
    },
    [handleSend, turnActive, sendInterrupt],
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

  const isTerminal = agentStatus !== null && TERMINAL_STATUSES.has(agentStatus);
  const canSend = status === "connected" && !isTerminal;
  const canInterrupt = canSend && turnActive;

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
          {hasDiff && (
            <div className={styles.viewToggle} role="group" aria-label="View mode">
              <button
                className={`${styles.toggleBtn} ${viewMode === "chat" ? styles.toggleActive : ""}`}
                onClick={() => setViewMode("chat")}
                aria-pressed={viewMode === "chat"}
              >
                Chat
              </button>
              <button
                className={`${styles.toggleBtn} ${viewMode === "diff" ? styles.toggleActive : ""}`}
                onClick={() => setViewMode("diff")}
                aria-pressed={viewMode === "diff"}
              >
                Diff
              </button>
            </div>
          )}
        </div>
        <div className={styles.headerActions}>
          {canInterrupt && (
            <button className={styles.interruptBtn} onClick={sendInterrupt} title="Interrupt (Esc)">
              &#x25A0;
            </button>
          )}
          <span className={styles.shortcutHint} title="Keyboard shortcuts: ⌘?">?</span>
          {onMinimize && (
            <button className={styles.minimizeBtn} onClick={onMinimize} title="Minimize (⌘⇧M)">
              &#x2013;
            </button>
          )}
          <button className={styles.closeBtn} onClick={onClose} title="Close (⌘⇧W)">
            &times;
          </button>
        </div>
      </div>

      {viewMode === "diff" && hasDiff ? (
        <div className={styles.diffArea}>
          <AgentDiffPanel slug={slug} branch={branch} />
        </div>
      ) : (
        <>
          <div className={styles.messages}>
            {messages.length === 0 && status === "connecting" && (
              <p className={styles.emptyState}>Connecting to agent...</p>
            )}
            {messages.length === 0 && status === "connected" && (
              <ThinkingIndicator />
            )}
            {messages.length === 0 && status === "disconnected" && isTerminal && (
              <p className={styles.emptyState}>{agentStatus === "suspended" ? "Agent suspended — no active connections" : `Agent ${agentStatus}`}</p>
            )}
            {messages.length === 0 && status === "reconnecting" && (
              <p className={styles.emptyState}>Reconnecting...</p>
            )}
            {messages.map((msg, i) => {
              if (msg.type === "turn_start") turnCounter++;
              return <MessageErrorBoundary key={i}><MessageDispatch msg={msg} turnNumber={turnCounter} /></MessageErrorBoundary>;
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
              placeholder={canSend ? (turnActive ? "Type to queue a message... (Esc to interrupt)" : "Type a message... (Enter to send)") : isTerminal ? (agentStatus === "suspended" ? "Agent suspended — resume to continue" : "Agent has exited") : "Connecting..."}
              disabled={!canSend}
              rows={1}
            />
            {queueDepth > 0 && (
              <span className={styles.queueBadge}>{queueDepth} queued</span>
            )}
            <button className={styles.sendBtn} onClick={handleSend} disabled={!canSend || !input.trim()}>
              {turnActive ? "Queue" : "Send"}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
