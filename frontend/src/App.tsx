import { useState, useMemo, useCallback } from "react";
import { Routes, Route, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { Agent, StatusResponse } from "./types/api";
import { useSSE } from "./hooks/useSSE";
import { useAgents } from "./hooks/useAgents";
import { useQuota } from "./hooks/useQuota";
import { useTracks } from "./hooks/useTracks";
import { useProjects } from "./hooks/useProjects";
import { useConsent } from "./hooks/useConsent";
import { useSkillsPrompt } from "./hooks/useSkillsPrompt";
import { queryKeys } from "./api/queryKeys";
import { fetcher, FetchError } from "./api/fetcher";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { AgentHistogram } from "./components/AgentHistogram";
import { LogViewer } from "./components/LogViewer";
import { AgentTerminal } from "./components/AgentTerminal";
import { SkillsBanner } from "./components/SkillsBanner";
import { ConsentDialog } from "./components/ConsentDialog";
import { SkillsInstallDialog } from "./components/SkillsInstallDialog";
import { ToastContainer } from "./components/toast/ToastContainer";
import { TourProvider } from "./components/tour/TourProvider";
import { TourOverlay, TourComplete } from "./components/tour/TourOverlay";
import { OverviewPage } from "./pages/OverviewPage";
import { AgentDetailPage } from "./pages/AgentDetailPage";
import { AgentHistoryPage } from "./pages/AgentHistoryPage";
import { ProjectPage } from "./pages/ProjectPage";
import { TracePage } from "./pages/TracePage";
import { TrackDetailPage } from "./pages/TrackDetailPage";
import styles from "./App.module.css";

export default function App() {
  const { agents, loading: agentsLoading, handleAgentUpdate, handleAgentRemoved } = useAgents();
  const { quota, handleQuotaUpdate } = useQuota();
  const { tracks, handleTrackUpdate, handleTrackRemoved } = useTracks();
  const { handleProjectUpdate, handleProjectRemoved } = useProjects();
  const { data: status = null } = useQuery({
    queryKey: queryKeys.status,
    queryFn: () => fetcher<StatusResponse>("/api/status"),
  });
  const [logAgentId, setLogAgentId] = useState<string | null>(null);
  const [terminalAgentId, setTerminalAgentId] = useState<string | null>(null);
  const consent = useConsent();
  const skillsPrompt = useSkillsPrompt();
  const queryClient = useQueryClient();

  const handleBoardUpdate = useCallback(
    () => {
      queryClient.invalidateQueries({ queryKey: ["board"] });
    },
    [queryClient],
  );

  const sseHandlers = useMemo(
    () => ({
      agent_update: handleAgentUpdate,
      agent_removed: handleAgentRemoved,
      quota_update: handleQuotaUpdate,
      track_update: handleTrackUpdate,
      track_removed: handleTrackRemoved,
      project_update: handleProjectUpdate,
      project_removed: handleProjectRemoved,
      board_update: handleBoardUpdate,
    }),
    [handleAgentUpdate, handleAgentRemoved, handleQuotaUpdate, handleTrackUpdate, handleTrackRemoved, handleProjectUpdate, handleProjectRemoved, handleBoardUpdate],
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

  const spawnMutation = useMutation({
    mutationFn: () =>
      fetcher<Agent>("/api/agents/interactive", { method: "POST" }),
    onSuccess: (agent) => {
      setTerminalAgentId(agent.id);
    },
    onError: (err) => {
      if (err instanceof FetchError && err.status === 403) {
        consent.requestConsent(() => spawnMutation.mutate());
      } else if (err instanceof FetchError && err.status === 412) {
        skillsPrompt.requestInstall(() => spawnMutation.mutate());
      }
    },
  });

  const handleSpawnInteractive = useCallback(() => {
    spawnMutation.mutate();
  }, [spawnMutation]);

  return (
    <TourProvider>
      <ToastContainer />
      <TourOverlay />
      <SkillsBanner />
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <h1 className={styles.title}>kiloforge</h1>
          <ConnectionStatus state={connectionState} />
          <AgentHistogram agents={agents} />
        </div>
        <nav>
          <Link to="/agents" className={styles.link}>Agents</Link>
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
                spawningInteractive={spawnMutation.isPending}
              />
            }
          />
          <Route path="/agents" element={<AgentHistoryPage />} />
          <Route path="/agents/:id" element={<AgentDetailPage />} />
          <Route path="/projects/:slug" element={<ProjectPage />} />
          <Route path="/projects/:slug/tracks/:trackId" element={<TrackDetailPage />} />
          <Route path="/traces/:traceId" element={<TracePage />} />
        </Routes>
      </main>

      {logAgentId && <LogViewer agentId={logAgentId} onClose={handleCloseLog} />}
      {terminalAgentId && <AgentTerminal agentId={terminalAgentId} onClose={handleCloseTerminal} />}
      {consent.showDialog && <ConsentDialog onAccept={consent.accept} onDeny={consent.deny} />}
      {skillsPrompt.showDialog && (
        <SkillsInstallDialog
          updating={skillsPrompt.updating}
          error={skillsPrompt.error}
          onInstall={skillsPrompt.install}
          onCancel={skillsPrompt.cancel}
        />
      )}
      <TourComplete />
    </TourProvider>
  );
}
