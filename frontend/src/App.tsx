import { useState, useEffect, useMemo, useCallback } from "react";
import { Routes, Route } from "react-router-dom";
import type { StatusResponse } from "./types/api";
import { useSSE } from "./hooks/useSSE";
import { useAgents } from "./hooks/useAgents";
import { useQuota } from "./hooks/useQuota";
import { useTracks } from "./hooks/useTracks";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { AgentHistogram } from "./components/AgentHistogram";
import { LogViewer } from "./components/LogViewer";
import { SkillsBanner } from "./components/SkillsBanner";
import { OverviewPage } from "./pages/OverviewPage";
import { ProjectPage } from "./pages/ProjectPage";
import { TracePage } from "./pages/TracePage";
import styles from "./App.module.css";

export default function App() {
  const { agents, loading: agentsLoading, handleAgentUpdate, handleAgentRemoved } = useAgents();
  const { quota, handleQuotaUpdate } = useQuota();
  const { tracks, handleTrackUpdate, handleTrackRemoved } = useTracks();
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [logAgentId, setLogAgentId] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/status")
      .then((r) => r.json())
      .then((data: StatusResponse) => setStatus(data))
      .catch(() => {});
  }, []);

  const sseHandlers = useMemo(
    () => ({
      agent_update: handleAgentUpdate,
      agent_removed: handleAgentRemoved,
      quota_update: handleQuotaUpdate,
      track_update: handleTrackUpdate,
      track_removed: handleTrackRemoved,
    }),
    [handleAgentUpdate, handleAgentRemoved, handleQuotaUpdate, handleTrackUpdate, handleTrackRemoved],
  );

  const connectionState = useSSE("/events", sseHandlers);

  const handleViewLog = useCallback((agentId: string) => {
    setLogAgentId(agentId);
  }, []);

  const handleCloseLog = useCallback(() => {
    setLogAgentId(null);
  }, []);

  return (
    <>
      <SkillsBanner />
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <h1 className={styles.title}>kiloforge</h1>
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
        <Routes>
          <Route
            path="/"
            element={
              <OverviewPage
                agents={agents}
                agentsLoading={agentsLoading}
                quota={quota}
                status={status}
                tracks={tracks}
                onViewLog={handleViewLog}
              />
            }
          />
          <Route path="/projects/:slug" element={<ProjectPage />} />
          <Route path="/traces/:traceId" element={<TracePage />} />
        </Routes>
      </main>

      {logAgentId && <LogViewer agentId={logAgentId} onClose={handleCloseLog} />}
    </>
  );
}
