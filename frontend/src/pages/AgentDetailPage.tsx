import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, Link, useNavigate, useLocation } from "react-router-dom";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { useQuery } from "@tanstack/react-query";
import type { Agent, LogResponse } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { StatusBadge } from "../components/StatusBadge";
import { InlineSpinner } from "../components/InlineSpinner";
import { formatUSD, formatTokens, formatUptime } from "../utils/format";
import { useAgentWebSocket } from "../hooks/useAgentWebSocket";
import type { WSConnectionState } from "../hooks/useAgentWebSocket";
import { MessageDispatch, MessageErrorBoundary } from "../components/terminal";
import { DiffView } from "../components/diff/DiffView";
import { useTracks } from "../hooks/useTracks";
import { useAgentActions, canStop, canResume, canReplace, canDelete } from "../hooks/useAgentActions";
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

interface AgentActionBarProps {
  agent: Agent;
  stop: ReturnType<typeof useAgentActions>["stop"];
  resume: ReturnType<typeof useAgentActions>["resume"];
  replace: ReturnType<typeof useAgentActions>["replace"];
  del: ReturnType<typeof useAgentActions>["del"];
  onShowReplace: () => void;
  onShowDelete: () => void;
}

function AgentActionBar({ agent, stop, resume, replace, del, onShowReplace, onShowDelete }: AgentActionBarProps) {
  return (
    <div className={styles.actionBar}>
      {canStop(agent) && (
        <button className={`${styles.actionBtn} ${styles.actionDanger}`} onClick={() => stop.mutate(agent.id)} disabled={stop.isPending}>
          {stop.isPending ? "Stopping..." : "Stop"}
        </button>
      )}
      {canResume(agent) && (
        <button className={`${styles.actionBtn} ${styles.actionSuccess}`} onClick={() => resume.mutate(agent.id)} disabled={resume.isPending}>
          {resume.isPending ? "Resuming..." : "Resume"}
        </button>
      )}
      {canReplace(agent) && (
        <button className={`${styles.actionBtn} ${styles.actionWarning}`} onClick={onShowReplace} disabled={replace.isPending}>
          {replace.isPending ? "Replacing..." : "Replace"}
        </button>
      )}
      {canDelete(agent) && (
        <button className={`${styles.actionBtn} ${styles.actionDanger}`} onClick={onShowDelete} disabled={del.isPending}>
          {del.isPending ? "Deleting..." : "Delete"}
        </button>
      )}
    </div>
  );
}

function MetaItem({ label, show = true, children }: { label: string; show?: boolean; children: React.ReactNode }) {
  if (!show) return null;
  return (
    <div className={styles.metaItem}>
      <span className={styles.metaLabel}>{label}</span>
      {children}
    </div>
  );
}

function TokenDisplay({ agent }: { agent: Agent }) {
  const cacheRead = agent.cache_read_tokens ?? 0;
  const cacheCreate = agent.cache_creation_tokens ?? 0;
  const hasCache = cacheRead > 0 || cacheCreate > 0;
  return (
    <span className={styles.mono}>
      {formatTokens(agent.input_tokens ?? 0)} in / {formatTokens(agent.output_tokens ?? 0)} out
      {hasCache && (
        <span className={styles.cacheInfo}>
          {" "}({formatTokens(cacheRead)} cache
          {cacheCreate > 0 && <>, {formatTokens(cacheCreate)} create</>})
        </span>
      )}
    </span>
  );
}

function AgentMetaGrid({ agent, projectSlug }: { agent: Agent; projectSlug: string | null }) {
  const hasTokens = (agent.input_tokens ?? 0) > 0 || (agent.output_tokens ?? 0) > 0;
  const shutdownLabel = agent.shutdown_reason === "idle_disconnect" ? "Suspended — no active connections" : agent.shutdown_reason;

  return (
    <div className={styles.metaGrid}>
      <MetaItem label="Role"><span className={`${styles.roleBadge} ${styles[agent.role] ?? ""}`}>{agent.role}</span></MetaItem>
      <MetaItem label="Status"><StatusBadge status={agent.status} /></MetaItem>
      <MetaItem label="Model" show={!!agent.model}><span>{agent.model}</span></MetaItem>
      <MetaItem label="Track" show={!!agent.ref}>
        <span className={styles.refValue}>
          {agent.ref}
          {projectSlug && <>{" "}<Link to={`/projects/${projectSlug}`} className={styles.boardLink}>View on Board</Link></>}
        </span>
      </MetaItem>
      <MetaItem label="Uptime" show={agent.uptime_seconds != null}><span>{formatUptime(agent.uptime_seconds ?? 0)}</span></MetaItem>
      <MetaItem label="PID" show={agent.pid > 0}><span className={styles.mono}>{agent.pid}</span></MetaItem>
      <MetaItem label="Worktree" show={!!agent.worktree_dir}><span className={styles.mono}>{agent.worktree_dir}</span></MetaItem>
      <MetaItem label="Tokens" show={hasTokens}><TokenDisplay agent={agent} /></MetaItem>
      <MetaItem label="Cost" show={agent.estimated_cost_usd != null}><span>{formatUSD(agent.estimated_cost_usd ?? 0)}</span></MetaItem>
      <MetaItem label="Suspended At" show={!!agent.suspended_at}><span>{agent.suspended_at ? new Date(agent.suspended_at).toLocaleString() : ""}</span></MetaItem>
      <MetaItem label="Shutdown Reason" show={!!agent.shutdown_reason}><span>{shutdownLabel}</span></MetaItem>
      <MetaItem label="Resume Error" show={!!agent.resume_error}><span className={styles.errorText}>{agent.resume_error}</span></MetaItem>
    </div>
  );
}

interface LogSectionProps {
  id: string;
}

function LogSection({ id }: LogSectionProps) {
  const [logLines, setLogLines] = useState<string[]>([]);
  const [logLoading, setLogLoading] = useState(true);
  const [following, setFollowing] = useState(false);
  const logRef = useRef<HTMLPreElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
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

  useEffect(() => {
    if (!following) {
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

  return (
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
  );
}

function AgentConfirmDialogs({ agent, replace, del, showReplace, showDelete, onHideReplace, onHideDelete }: {
  agent: Agent;
  replace: ReturnType<typeof useAgentActions>["replace"];
  del: ReturnType<typeof useAgentActions>["del"];
  showReplace: boolean;
  showDelete: boolean;
  onHideReplace: () => void;
  onHideDelete: () => void;
}) {
  const navigate = useNavigate();
  return (
    <>
      {showReplace && (
        <ConfirmDialog
          title="Replace Agent"
          message="This agent's session could not be recovered. Replace with a new agent for the same work?"
          confirmLabel="Replace"
          confirming={replace.isPending}
          onConfirm={() => replace.mutate(agent.id, {
            onSuccess: (newAgent) => navigate(`/agents/${newAgent.id}`),
          })}
          onCancel={onHideReplace}
        />
      )}
      {showDelete && (
        <ConfirmDialog
          title="Delete Agent"
          message={`Are you sure you want to delete "${agent.name || agent.id}"?`}
          confirmLabel="Delete"
          confirming={del.isPending}
          onConfirm={() => del.mutate(agent.id, { onSuccess: () => navigate("/") })}
          onCancel={onHideDelete}
        />
      )}
    </>
  );
}

function BranchDiffSection({ agent, projectSlug, diffRef }: {
  agent: Agent;
  projectSlug: string;
  diffRef: React.RefObject<HTMLDivElement | null>;
}) {
  return (
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
  );
}

export function AgentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const diffRef = useRef<HTMLDivElement>(null);

  const { tracks } = useTracks();
  const { stop, resume, replace, del } = useAgentActions();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showReplaceConfirm, setShowReplaceConfirm] = useState(false);

  const { data: agent, error: agentError } = useQuery({
    queryKey: queryKeys.agent(id ?? ""),
    queryFn: () => fetcher<Agent>(`/api/agents/${encodeURIComponent(id!)}`),
    enabled: !!id,
  });
  const error = agentError?.message ?? null;

  useEffect(() => {
    if (location.hash === "#diff") {
      diffRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [location.hash, agent]);

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

  const projectSlug = tracks.find((t) => t.id === agent.ref)?.project ?? null;

  return (
    <div className={styles.page}>
      <div className={styles.topBar}>
        <Link to="/" className={styles.back}>&larr; Back</Link>
        <h2 className={styles.title}>
          Agent <span className={styles.agentId}>{agent.name || agent.id}</span>
        </h2>
      </div>

      <AgentActionBar
        agent={agent} stop={stop} resume={resume} replace={replace} del={del}
        onShowReplace={() => setShowReplaceConfirm(true)}
        onShowDelete={() => setShowDeleteConfirm(true)}
      />

      <AgentConfirmDialogs
        agent={agent} replace={replace} del={del}
        showReplace={showReplaceConfirm} showDelete={showDeleteConfirm}
        onHideReplace={() => setShowReplaceConfirm(false)}
        onHideDelete={() => setShowDeleteConfirm(false)}
      />

      {agent.status === "replaced" && (
        <div className={styles.replacedBanner}>
          This agent has been replaced by a new agent for the same work.
        </div>
      )}

      <AgentMetaGrid agent={agent} projectSlug={projectSlug} />

      {agent.worktree_dir && projectSlug && (
        <BranchDiffSection agent={agent} projectSlug={projectSlug} diffRef={diffRef} />
      )}

      {id && <LogSection id={id} />}

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
      if (e.nativeEvent.isComposing) return;
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
          return <MessageErrorBoundary key={i}><MessageDispatch msg={msg} turnNumber={turnCounter} /></MessageErrorBoundary>;
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
