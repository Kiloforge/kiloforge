import { useState, useMemo, useCallback } from "react";
import { Routes, Route } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import type { Agent, StatusResponse } from "./types/api";
import { useSSE } from "./hooks/useSSE";
import { useAgents } from "./hooks/useAgents";
import { useQuota } from "./hooks/useQuota";
import { useTracks } from "./hooks/useTracks";
import { useConsent } from "./hooks/useConsent";
import { queryKeys } from "./api/queryKeys";
import { fetcher } from "./api/fetcher";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { AgentHistogram } from "./components/AgentHistogram";
import { LogViewer } from "./components/LogViewer";
import { AgentTerminal } from "./components/AgentTerminal";
import { SkillsBanner } from "./components/SkillsBanner";
import { ConsentDialog } from "./components/ConsentDialog";
import { OverviewPage } from "./pages/OverviewPage";
import { AgentDetailPage } from "./pages/AgentDetailPage";
import { ProjectPage } from "./pages/ProjectPage";
import { TracePage } from "./pages/TracePage";
import styles from "./App.module.css";

export default function App() {
  const { agents, loading: agentsLoading, handleAgentUpdate, handleAgentRemoved } = useAgents();
  const { quota, handleQuotaUpdate } = useQuota();
  const { tracks, handleTrackUpdate, handleTrackRemoved } = useTracks();
  const { data: status = null } = useQuery({
    queryKey: queryKeys.status,
    queryFn: () => fetcher<StatusResponse>("/api/status"),
  });
  const [logAgentId, setLogAgentId] = useState<string | null>(null);
  const [terminalAgentId, setTerminalAgentId] = useState<string | null>(null);
  const [spawningInteractive, setSpawningInteractive] = useState(false);
  const consent = useConsent();

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

  const handleAttach = useCallback((agentId: string) => {
    setTerminalAgentId(agentId);
  }, []);

  const handleCloseTerminal = useCallback(() => {
    setTerminalAgentId(null);
  }, []);

  const handleSpawnInteractive = useCallback(async () => {
    setSpawningInteractive(true);
    try {
      const res = await fetch("/api/agents/interactive", { method: "POST" });
      if (res.status === 403) {
        consent.requestConsent(() => handleSpawnInteractive());
        return;
      }
      if (!res.ok) throw new Error("Failed to spawn");
      const agent = (await res.json()) as Agent;
      setTerminalAgentId(agent.id);
    } catch {
      // silent fail — user sees the button re-enable
    } finally {
      setSpawningInteractive(false);
    }
  }, [consent]);

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
            <a href="/gitea/" target="_blank" rel="noopener noreferrer" className={styles.link}>
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
                tracks={tracks}
                onViewLog={handleViewLog}
                onAttach={handleAttach}
                onSpawnInteractive={handleSpawnInteractive}
                spawningInteractive={spawningInteractive}
              />
            }
          />
          <Route path="/agents/:id" element={<AgentDetailPage />} />
          <Route path="/projects/:slug" element={<ProjectPage />} />
          <Route path="/traces/:traceId" element={<TracePage />} />
        </Routes>
      </main>

      {logAgentId && <LogViewer agentId={logAgentId} onClose={handleCloseLog} />}
      {terminalAgentId && <AgentTerminal agentId={terminalAgentId} onClose={handleCloseTerminal} />}
      {consent.showDialog && <ConsentDialog onAccept={consent.accept} onDeny={consent.deny} />}
    </>
  );
}
