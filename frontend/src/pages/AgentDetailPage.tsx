import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, Link, useNavigate, useLocation } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import type { Agent, LogResponse } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { StatusBadge } from "../components/StatusBadge";
import { InlineSpinner } from "../components/InlineSpinner";
import { formatUSD, formatTokens, formatUptime } from "../utils/format";
import { useAgentWebSocket } from "../hooks/useAgentWebSocket";
import type { WSConnectionState } from "../hooks/useAgentWebSocket";
import { MessageDispatch } from "../components/terminal";
import { DiffView } from "../components/diff/DiffView";
import { useTracks } from "../hooks/useTracks";
import { useAgentActions, canStop, canResume, canDelete } from "../hooks/useAgentActions";
import styles from "./AgentDetailPage.module.css";

function ConnectionDot({ status }: { status: WSConnectionState }) {
  const cls =
    status === "connected"
      ? styles.dotConnected
      : status === "reconnecting" || status === "connecting"
        ? styles.dotReconnecting
        : styles.dotDisconnected;
  return <span className={`${styles.dot} ${cls}`} />;
}

export function AgentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const diffRef = useRef<HTMLDivElement>(null);

  const { tracks } = useTracks();
  const { stop, resume, del } = useAgentActions();

  const { data: agent, error: agentError } = useQuery({
    queryKey: queryKeys.agent(id ?? ""),
    queryFn: () => fetcher<Agent>(`/api/agents/${encodeURIComponent(id!)}`),
    enabled: !!id,
  });
  const error = agentError?.message ?? null;

  // Log viewer state
  const [logLines, setLogLines] = useState<string[]>([]);
  const [logLoading, setLogLoading] = useState(true);
  const [following, setFollowing] = useState(false);
  const logRef = useRef<HTMLPreElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Scroll to diff section if hash is #diff
  useEffect(() => {
    if (location.hash === "#diff") {
      diffRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [location.hash, agent]);

  // Fetch log data (keep as raw fetch — streaming log is not cache-friendly)
  useEffect(() => {
    if (!id) {
      setLogLoading(false);
      return;
    }
    let cancelled = false;
    setLogLoading(true);
    fetch(`/api/agents/${encodeURIComponent(id)}/log?lines=200`)
      .then((r) => r.json())
      .then((data: LogResponse) => {
        if (cancelled) return;
        setLogLines(data.lines || []);
        setLogLoading(false);
        requestAnimationFrame(() => {
          if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
        });
      })
      .catch(() => {
        if (cancelled) return;
        setLogLines(["Failed to load log."]);
        setLogLoading(false);
      });
    return () => { cancelled = true; };
  }, [id]);

  // Follow mode
  useEffect(() => {
    if (!id || !following) {
      eventSourceRef.current?.close();
      eventSourceRef.current = null;
      return;
    }
    const es = new EventSource(`/api/agents/${encodeURIComponent(id)}/log?lines=200&follow=true`);
    eventSourceRef.current = es;
    es.onmessage = (e) => {
      setLogLines((prev) => [...prev, e.data as string]);
      requestAnimationFrame(() => {
        if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
      });
    };
    es.onerror = () => { es.close(); setFollowing(false); };
    return () => { es.close(); };
  }, [following, id]);

  useEffect(() => {
    return () => { eventSourceRef.current?.close(); };
  }, []);

  if (error) {
    return (
      <div className={styles.page}>
        <Link to="/" className={styles.back}>&larr; Back</Link>
        <p className={styles.error}>{error}</p>
      </div>
    );
  }

  if (!agent) {
    return (
      <div className={styles.page}>
        <Link to="/" className={styles.back}>&larr; Back</Link>
        <InlineSpinner label="Loading agent..." />
      </div>
    );
  }

  const hasTokens = (agent.input_tokens ?? 0) > 0 || (agent.output_tokens ?? 0) > 0;
  const cacheRead = agent.cache_read_tokens ?? 0;
  const cacheCreate = agent.cache_creation_tokens ?? 0;
  const matchedTrack = agent.ref ? tracks.find((t) => t.id === agent.ref) : null;
  const projectSlug = matchedTrack?.project ?? null;

  return (
    <div className={styles.page}>
      <div className={styles.topBar}>
        <Link to="/" className={styles.back}>&larr; Back</Link>
        <h2 className={styles.title}>
          Agent <span className={styles.agentId}>{agent.name || agent.id}</span>
        </h2>
      </div>

      <div className={styles.actionBar}>
        {canStop(agent) && (
          <button
            className={`${styles.actionBtn} ${styles.actionDanger}`}
            onClick={() => stop.mutate(agent.id)}
            disabled={stop.isPending}
          >
            {stop.isPending ? "Stopping..." : "Stop"}
          </button>
        )}
        {canResume(agent) && (
          <button
            className={`${styles.actionBtn} ${styles.actionSuccess}`}
            onClick={() => resume.mutate(agent.id)}
            disabled={resume.isPending}
          >
            {resume.isPending ? "Resuming..." : "Resume"}
          </button>
        )}
        {canDelete(agent) && (
          <button
            className={`${styles.actionBtn} ${styles.actionDanger}`}
            onClick={() => {
              if (window.confirm(`Delete agent "${agent.name || agent.id}"?`)) {
                del.mutate(agent.id, {
                  onSuccess: () => navigate("/"),
                });
              }
            }}
            disabled={del.isPending}
          >
            {del.isPending ? "Deleting..." : "Delete"}
          </button>
        )}
      </div>

      <div className={styles.metaGrid}>
        <div className={styles.metaItem}>
          <span className={styles.metaLabel}>Role</span>
          <span className={`${styles.roleBadge} ${styles[agent.role] ?? ""}`}>{agent.role}</span>
        </div>
        <div className={styles.metaItem}>
          <span className={styles.metaLabel}>Status</span>
          <StatusBadge status={agent.status} />
        </div>
        {agent.model && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Model</span>
            <span>{agent.model}</span>
          </div>
        )}
        {agent.ref && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Track</span>
            <span className={styles.refValue}>
              {agent.ref}
              {projectSlug && (
                <>
                  {" "}
                  <Link to={`/projects/${projectSlug}`} className={styles.boardLink}>View on Board</Link>
                </>
              )}
            </span>
          </div>
        )}
        {agent.uptime_seconds != null && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Uptime</span>
            <span>{formatUptime(agent.uptime_seconds)}</span>
          </div>
        )}
        {agent.pid > 0 && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>PID</span>
            <span className={styles.mono}>{agent.pid}</span>
          </div>
        )}
        {agent.worktree_dir && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Worktree</span>
            <span className={styles.mono}>{agent.worktree_dir}</span>
          </div>
        )}
        {hasTokens && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Tokens</span>
            <span className={styles.mono}>
              {formatTokens(agent.input_tokens ?? 0)} in / {formatTokens(agent.output_tokens ?? 0)} out
              {(cacheRead > 0 || cacheCreate > 0) && (
                <span className={styles.cacheInfo}>
                  {" "}({formatTokens(cacheRead)} cache
                  {cacheCreate > 0 && <>, {formatTokens(cacheCreate)} create</>})
                </span>
              )}
            </span>
          </div>
        )}
        {agent.estimated_cost_usd != null && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Cost</span>
            <span>{formatUSD(agent.estimated_cost_usd)}</span>
          </div>
        )}
      </div>

      {agent.worktree_dir && projectSlug && (
        <div ref={diffRef} className={styles.diffSection} id="diff">
          <h3 className={styles.sectionTitle}>Branch Diff</h3>
          <DiffView
            slug={projectSlug}
            branch={agent.ref}
            onDiscuss={agent.role === "interactive" ? () => {
              const termEl = document.getElementById("terminal");
              termEl?.scrollIntoView({ behavior: "smooth" });
            } : undefined}
          />
        </div>
      )}

      <div className={styles.logSection}>
        <div className={styles.logHeader}>
          <h3>Log Output</h3>
          <label className={styles.followToggle}>
            <input type="checkbox" checked={following} onChange={(e) => setFollowing(e.target.checked)} />
            Follow
          </label>
        </div>
        <pre ref={logRef} className={styles.logViewer}>
          {logLoading ? <InlineSpinner label="Loading log..." /> : logLines.join("\n") || "No log data available."}
        </pre>
      </div>

      {agent.role === "interactive" && id && <div id="terminal"><TerminalSection agentId={id} /></div>}
    </div>
  );
}

function TerminalSection({ agentId }: { agentId: string }) {
  const { messages, sendMessage, status, agentStatus } = useAgentWebSocket(agentId);
  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

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
  const canSend = status === "connected" && !isTerminal;

  let turnCounter = 0;

  return (
    <div className={styles.terminalSection}>
      <div className={styles.terminalHeader}>
        <h3>Terminal</h3>
        <ConnectionDot status={status} />
      </div>
      <div className={styles.terminalMessages}>
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
      <div className={styles.terminalInput}>
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
