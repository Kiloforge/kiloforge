import { useState, useMemo, useCallback, useEffect } from "react";
import { Routes, Route, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { Agent, SpawnInteractiveRequest, StatusResponse, SwarmCapacity } from "./types/api";
import type { AgentRole } from "./components/AgentLauncher";
import { useSSE } from "./hooks/useSSE";
import { useAgents } from "./hooks/useAgents";
import { useQuota } from "./hooks/useQuota";
import { useTracks } from "./hooks/useTracks";
import { useProjects } from "./hooks/useProjects";
import { useSwarm } from "./hooks/useSwarm";
import { useSwarmCapacity } from "./hooks/useSwarmCapacity";
import { useConsent } from "./hooks/useConsent";
import { useSkillsPrompt } from "./hooks/useSkillsPrompt";
import { queryKeys } from "./api/queryKeys";
import { fetcher, FetchError } from "./api/fetcher";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { AgentHistogram } from "./components/AgentHistogram";
import { LogViewer } from "./components/LogViewer";
import { AgentTerminal } from "./components/AgentTerminal";
import { TerminalDock } from "./components/TerminalDock";
import { useWindowManager } from "./hooks/useWindowManager";
import { useKeyboardShortcuts } from "./hooks/useKeyboardShortcuts";
import { ShortcutHelp } from "./components/ShortcutHelp";
import { SkillsBanner } from "./components/SkillsBanner";
import { ModelWarningBanner } from "./components/ModelWarningBanner";
import { ConsentDialog } from "./components/ConsentDialog";
import { SkillsInstallDialog } from "./components/SkillsInstallDialog";
import { AgentLauncher } from "./components/AgentLauncher";
import { ToastContainer } from "./components/toast/ToastContainer";
import { TourProvider } from "./components/tour/TourProvider";
import { TourOverlay } from "./components/tour/TourOverlay";
import { SettingsMenu } from "./components/SettingsMenu";
import { OverviewPage } from "./pages/OverviewPage";
import { AgentDetailPage } from "./pages/AgentDetailPage";
import { AgentHistoryPage } from "./pages/AgentHistoryPage";
import { ProjectPage } from "./pages/ProjectPage";
import { TracePage } from "./pages/TracePage";
import { TrackDetailPage } from "./pages/TrackDetailPage";
import styles from "./App.module.css";

export default function App() {
  const { agents, loading: agentsLoading, handleAgentUpdate, handleAgentRemoved, remainingCount: agentRemainingCount, hasNextPage: agentHasNextPage, isFetchingNextPage: agentFetchingNextPage, fetchNextPage: agentFetchNextPage } = useAgents();
  const { quota, handleQuotaUpdate } = useQuota();
  const { tracks, handleTrackUpdate, handleTrackRemoved, remainingCount: trackRemainingCount, hasNextPage: trackHasNextPage, isFetchingNextPage: trackFetchingNextPage, fetchNextPage: trackFetchNextPage } = useTracks();
  const { handleProjectUpdate, handleProjectRemoved } = useProjects();
  const { swarm, loading: swarmLoading, starting: swarmStarting, stopping: swarmStopping, updatingSettings: swarmUpdatingSettings, start: swarmStart, stop: swarmStop, updateSettings: swarmUpdateSettings, handleSwarmUpdate } = useSwarm();
  const { data: status = null } = useQuery({
    queryKey: queryKeys.status,
    queryFn: () => fetcher<StatusResponse>("/api/status"),
  });
  const { capacity, handleCapacityChanged } = useSwarmCapacity();
  const [logAgentId, setLogAgentId] = useState<string | null>(null);
  const wm = useWindowManager();
  const [showLauncher, setShowLauncher] = useState(false);
  const [waitingForCapacity, setWaitingForCapacity] = useState(false);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const consent = useConsent();
  const skillsPrompt = useSkillsPrompt();
  const queryClient = useQueryClient();

  const shortcutActions = useMemo(
    () => ({
      tileAll: () => wm.tileAll(),
      cycleFocusNext: () => wm.cycleFocus(1),
      cycleFocusPrev: () => wm.cycleFocus(-1),
      toggleMinimize: () => wm.toggleMinimizeFocused(),
      toggleMaximize: () => wm.toggleMaximizeFocused(),
      closeFocused: () => wm.closeFocused(),
      snapLeft: () => wm.snapFocused("left"),
      snapRight: () => wm.snapFocused("right"),
      showHelp: () => setShowShortcuts((v) => !v),
    }),
    [wm],
  );

  useKeyboardShortcuts(shortcutActions);

  const handleBoardUpdate = useCallback(
    () => {
      queryClient.invalidateQueries({ queryKey: ["board"] });
    },
    [queryClient],
  );

  const handleProjectSettingsUpdate = useCallback(
    () => {
      queryClient.invalidateQueries({ queryKey: ["projectSettings"] });
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
      queue_update: handleSwarmUpdate,
      capacity_changed: handleCapacityChanged,
      project_settings_update: handleProjectSettingsUpdate,
    }),
    [handleAgentUpdate, handleAgentRemoved, handleQuotaUpdate, handleTrackUpdate, handleTrackRemoved, handleProjectUpdate, handleProjectRemoved, handleBoardUpdate, handleSwarmUpdate, handleCapacityChanged, handleProjectSettingsUpdate],
  );

  const connectionState = useSSE("/events", sseHandlers);

  const handleViewLog = useCallback((agentId: string) => {
    setLogAgentId(agentId);
  }, []);

  const handleCloseLog = useCallback(() => {
    setLogAgentId(null);
  }, []);

  const handleAttach = useCallback((agentId: string) => {
    if (wm.has(agentId)) return; // already open — z-index handled by panel click
    const agent = agents.find((a) => a.id === agentId);
    wm.open(agentId, agent?.name, agent?.role);
  }, [wm, agents]);

  const spawnMutation = useMutation({
    mutationFn: (req: SpawnInteractiveRequest) =>
      fetcher<Agent>("/api/agents/interactive", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }),
    onSuccess: (agent) => {
      wm.open(agent.id, agent.name, agent.role);
      setShowLauncher(false);
    },
    onError: (err) => {
      if (err instanceof FetchError && err.status === 403) {
        consent.requestConsent(() => spawnMutation.mutate(lastSpawnReq));
      } else if (err instanceof FetchError && err.status === 412) {
        skillsPrompt.requestInstall(() => spawnMutation.mutate(lastSpawnReq));
      } else if (err instanceof FetchError && err.status === 429) {
        setWaitingForCapacity(true);
      }
    },
  });

  const [lastSpawnReq, setLastSpawnReq] = useState<SpawnInteractiveRequest>({});

  const handleOpenLauncher = useCallback(() => {
    setShowLauncher(true);
  }, []);

  const handleCloseLauncher = useCallback(() => {
    setShowLauncher(false);
  }, []);

  const handleLaunch = useCallback((role: AgentRole, prompt: string) => {
    const req: SpawnInteractiveRequest = { role };
    if (prompt) req.prompt = prompt;
    setLastSpawnReq(req);
    spawnMutation.mutate(req);
  }, [spawnMutation]);

  // Auto-retry spawn when capacity becomes available
  useEffect(() => {
    if (waitingForCapacity && capacity && capacity.available > 0) {
      setWaitingForCapacity(false);
      setWaitingCapacity(null);
      spawnMutation.mutate(lastSpawnReq);
    }
  }, [waitingForCapacity, capacity, lastSpawnReq, spawnMutation]);

  // Derive displayed capacity from live data when waiting
  const waitingCapacity = waitingForCapacity ? (capacity ?? null) : null;

  const handleCancelWaiting = useCallback(() => {
    setWaitingForCapacity(false);
    setShowLauncher(false);
  }, []);

  return (
    <TourProvider>
      <ToastContainer />
      <TourOverlay />
      <ModelWarningBanner />
      <SkillsBanner />
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          <Link to="/" className={styles.homeLink}>
            <img src="/kf_logo.webp" alt="kiloforge" className={styles.logo} />
            <h1 className={styles.title}>kiloforge</h1>
            <span className={styles.subtitle}>Command Deck</span>
          </Link>
          <ConnectionStatus state={connectionState} />
          <span className={styles.headerDivider} />
          <span className={styles.headerLabel}>Agents</span>
          <AgentHistogram agents={agents} />
          {wm.count > 0 && (
            <>
              <span className={styles.headerDivider} />
              <span className={styles.headerLabel}>{wm.count} terminal{wm.count !== 1 ? "s" : ""}</span>
            </>
          )}
        </div>
        <nav className={styles.nav}>
          <Link to="/agents" className={styles.link}>Agents</Link>
          {status?.gitea_url && (
            <a href="/gitea/" target="_blank" rel="noopener noreferrer" className={styles.link}>
              Gitea
            </a>
          )}
          <SettingsMenu />
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
                agentRemainingCount={agentRemainingCount}
                agentHasNextPage={agentHasNextPage}
                agentFetchingNextPage={agentFetchingNextPage}
                onAgentLoadMore={agentFetchNextPage}
                quota={quota}
                tracks={tracks}
                onViewLog={handleViewLog}
                onAttach={handleAttach}
                onSpawnInteractive={handleOpenLauncher}
                spawningInteractive={spawnMutation.isPending}
                swarm={swarm}
                swarmLoading={swarmLoading}
                swarmStarting={swarmStarting}
                swarmStopping={swarmStopping}
                swarmUpdatingSettings={swarmUpdatingSettings}
                onSwarmStart={swarmStart}
                onSwarmStop={swarmStop}
                onSwarmUpdateSettings={swarmUpdateSettings}
                trackRemainingCount={trackRemainingCount}
                trackHasNextPage={trackHasNextPage}
                trackFetchingNextPage={trackFetchingNextPage}
                onTrackLoadMore={trackFetchNextPage}
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
      {wm.getWindows().map((entry) => (
        <AgentTerminal
          key={entry.agentId}
          agentId={entry.agentId}
          name={entry.name}
          role={entry.role}
          initialX={entry.initialX}
          initialY={entry.initialY}
          minimized={entry.minimized}
          onClose={() => wm.close(entry.agentId)}
          onMinimize={() => wm.minimize(entry.agentId)}
          onActivity={() => wm.incrementUnread(entry.agentId)}
          registerControls={wm.registerControls}
          unregisterControls={wm.unregisterControls}
        />
      ))}
      {showShortcuts && <ShortcutHelp onClose={() => setShowShortcuts(false)} />}
      <TerminalDock
        windows={wm.getMinimizedWindows()}
        onRestore={(id) => wm.restore(id)}
        onClose={(id) => wm.close(id)}
      />
      {(showLauncher || waitingForCapacity) && (
        <AgentLauncher
          onLaunch={handleLaunch}
          onClose={handleCloseLauncher}
          launching={spawnMutation.isPending}
          waitingForCapacity={waitingForCapacity}
          waitingCapacity={waitingCapacity}
          onCancelWaiting={handleCancelWaiting}
        />
      )}
      {consent.showDialog && <ConsentDialog onAccept={consent.accept} onDeny={consent.deny} />}
      {skillsPrompt.showDialog && (
        <SkillsInstallDialog
          updating={skillsPrompt.updating}
          error={skillsPrompt.error}
          onInstall={skillsPrompt.install}
          onCancel={skillsPrompt.cancel}
        />
      )}
    </TourProvider>
  );
}
