import { useState, useEffect, useMemo, useCallback } from "react";
import type { StatusResponse } from "./types/api";
import { useSSE } from "./hooks/useSSE";
import { useAgents } from "./hooks/useAgents";
import { useQuota } from "./hooks/useQuota";
import { useTracks } from "./hooks/useTracks";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { AgentHistogram } from "./components/AgentHistogram";
import { StatCards } from "./components/StatCards";
import { AgentGrid } from "./components/AgentGrid";
import { TrackList } from "./components/TrackList";
import { LogViewer } from "./components/LogViewer";
import styles from "./App.module.css";

export default function App() {
  const { agents, loading: agentsLoading, handleAgentUpdate, handleAgentRemoved } = useAgents();
  const { quota, handleQuotaUpdate } = useQuota();
  const { tracks } = useTracks();
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [logAgentId, setLogAgentId] = useState<string | null>(null);

  useEffect(() => {
    fetch("/-/api/status")
      .then((r) => r.json())
      .then((data: StatusResponse) => setStatus(data))
      .catch(() => {});
  }, []);

  const sseHandlers = useMemo(
    () => ({
      agent_update: handleAgentUpdate,
      agent_removed: handleAgentRemoved,
      quota_update: handleQuotaUpdate,
    }),
    [handleAgentUpdate, handleAgentRemoved, handleQuotaUpdate],
  );

  const connectionState = useSSE("/-/events", sseHandlers);

  const handleViewLog = useCallback((agentId: string) => {
    setLogAgentId(agentId);
  }, []);

  const handleCloseLog = useCallback(() => {
    setLogAgentId(null);
  }, []);

  return (
    <>
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <h1 className={styles.title}>crelay</h1>
          <ConnectionStatus state={connectionState} />
          <AgentHistogram agents={agents} />
        </div>
        <nav>
          {status?.gitea_url && (
            <a href="/" target="_blank" rel="noopener noreferrer" className={styles.link}>
              Gitea
            </a>
          )}
        </nav>
      </header>

      <main className={styles.main}>
        <StatCards agentCount={agents.length} quota={quota} />

        <section className={styles.panel}>
          <h2 className={styles.panelTitle}>Agents</h2>
          {agentsLoading ? (
            <p className={styles.empty}>Loading agents...</p>
          ) : (
            <AgentGrid agents={agents} giteaURL={status?.gitea_url ?? ""} onViewLog={handleViewLog} />
          )}
        </section>

        <section className={styles.panel}>
          <h2 className={styles.panelTitle}>Tracks</h2>
          <TrackList tracks={tracks} />
        </section>
      </main>

      {logAgentId && <LogViewer agentId={logAgentId} onClose={handleCloseLog} />}
    </>
  );
}
