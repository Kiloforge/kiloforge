import { useState } from "react";
import type { WSMessage } from "../../hooks/useAgentWebSocket";
import { MarkdownContent } from "./MarkdownContent";
import { formatTokens, formatUSD } from "../../utils/format";
import styles from "./TerminalBubbles.module.css";

/** Renders a user input message. */
function InputBubble({ msg }: { msg: WSMessage }) {
  return (
    <div className={`${styles.message} ${styles.userMessage}`}>
      <span className={styles.messageIcon}>you</span>
      <div className={styles.messageContent}>{msg.text}</div>
    </div>
  );
}

/** Renders a text content block from the agent. */
function TextBubble({ msg }: { msg: WSMessage }) {
  return (
    <div className={`${styles.message} ${styles.agentMessage}`}>
      <span className={styles.messageIcon}>kf</span>
      <div className={styles.messageContent}>
        <MarkdownContent text={msg.text} />
      </div>
    </div>
  );
}

/** Renders a tool invocation with collapsible JSON input. */
function ToolUseBubble({ msg }: { msg: WSMessage }) {
  const [expanded, setExpanded] = useState(false);
  const hasInput = msg.toolInput && Object.keys(msg.toolInput).length > 0;

  return (
    <div className={`${styles.message} ${styles.toolMessage}`}>
      <span className={styles.messageIcon}>tool</span>
      <div className={styles.messageContent}>
        <div className={styles.toolHeader}>
          <span className={styles.toolName}>{msg.toolName}</span>
          {msg.toolId && <span className={styles.toolId}>{msg.toolId}</span>}
          {hasInput && (
            <button
              className={styles.toggleBtn}
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? "Hide" : "Show"} input
            </button>
          )}
        </div>
        {expanded && hasInput && (
          <pre className={styles.toolInput}>
            {JSON.stringify(msg.toolInput, null, 2)}
          </pre>
        )}
      </div>
    </div>
  );
}

/** Renders a thinking block with collapsible content. */
function ThinkingBubble({ msg }: { msg: WSMessage }) {
  const [expanded, setExpanded] = useState(false);
  const text = msg.thinking ?? "";
  const isLong = text.length > 120;

  return (
    <div className={`${styles.message} ${styles.thinkingMessage}`}>
      <span className={styles.messageIcon}>think</span>
      <div className={styles.messageContent}>
        {isLong ? (
          <>
            <span className={styles.thinkingText}>
              {expanded ? <MarkdownContent text={text} /> : text.slice(0, 120) + "..."}
            </span>
            <button
              className={styles.toggleBtn}
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? "Less" : "More"}
            </button>
          </>
        ) : (
          <span className={styles.thinkingText}>
            <MarkdownContent text={text} />
          </span>
        )}
      </div>
    </div>
  );
}

/** Renders a turn boundary separator. */
function TurnSeparator({ turnNumber }: { turnNumber: number }) {
  return (
    <div className={styles.turnSeparator}>
      <span className={styles.turnLabel}>Turn {turnNumber}</span>
    </div>
  );
}

/** Renders a turn end summary with cost and token usage. */
function TurnEndSummary({ msg }: { msg: WSMessage }) {
  const hasUsage = msg.usage && (msg.usage.input_tokens > 0 || msg.usage.output_tokens > 0);
  const hasCost = msg.costUsd != null && msg.costUsd > 0;

  if (!hasUsage && !hasCost) return null;

  return (
    <div className={styles.turnEndSummary}>
      {hasUsage && msg.usage && (
        <span>
          {formatTokens(msg.usage.input_tokens)} in / {formatTokens(msg.usage.output_tokens)} out
          {msg.usage.cache_read_tokens > 0 && (
            <> ({formatTokens(msg.usage.cache_read_tokens)} cached)</>
          )}
        </span>
      )}
      {hasCost && <span>{formatUSD(msg.costUsd!)}</span>}
    </div>
  );
}

/** Renders a system notification with severity-based styling. */
function SystemBubble({ msg }: { msg: WSMessage }) {
  const subtype = msg.subtype ?? "info";
  const severityClass =
    subtype === "error" ? styles.systemError
    : subtype === "warning" ? styles.systemWarning
    : styles.systemInfo;

  const dataText = msg.data ? JSON.stringify(msg.data) : "";

  return (
    <div className={`${styles.message} ${styles.systemMessage} ${severityClass}`}>
      <span className={styles.messageIcon}>sys</span>
      <div className={styles.messageContent}>
        <span className={styles.systemSubtype}>{subtype}</span>
        {dataText && <span className={styles.systemData}>{dataText}</span>}
      </div>
    </div>
  );
}

/** Renders a status message (e.g., agent exited). */
function StatusBubble({ msg }: { msg: WSMessage }) {
  return (
    <div className={`${styles.message} ${styles.statusMessage}`}>
      <span className={styles.statusText}>{msg.text}</span>
    </div>
  );
}

/** Renders an error message. */
function ErrorBubble({ msg }: { msg: WSMessage }) {
  return (
    <div className={`${styles.message} ${styles.errorMessage}`}>
      <span className={styles.messageIcon}>err</span>
      <div className={styles.messageContent}>{msg.text}</div>
    </div>
  );
}

interface MessageDispatchProps {
  msg: WSMessage;
  turnNumber?: number;
}

/**
 * Dispatches a WSMessage to the appropriate bubble component.
 * turnNumber is used for turn_start separators.
 */
export function MessageDispatch({ msg, turnNumber }: MessageDispatchProps) {
  switch (msg.type) {
    case "input":
      return <InputBubble msg={msg} />;
    case "text":
    case "output":
      return <TextBubble msg={msg} />;
    case "tool_use":
      return <ToolUseBubble msg={msg} />;
    case "thinking":
      return <ThinkingBubble msg={msg} />;
    case "turn_start":
      return <TurnSeparator turnNumber={turnNumber ?? 1} />;
    case "turn_end":
      return <TurnEndSummary msg={msg} />;
    case "system":
      return <SystemBubble msg={msg} />;
    case "status":
      return <StatusBubble msg={msg} />;
    case "error":
      return <ErrorBubble msg={msg} />;
    default:
      return null;
  }
}
