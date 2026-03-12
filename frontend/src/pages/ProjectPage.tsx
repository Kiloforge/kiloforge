import { useCallback, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Agent, Project, ProjectMetadata, ResolveConflictRequest, SpawnInteractiveRequest } from "../types/api";
import type { AgentRole } from "../components/AgentLauncher";
import { useTracks } from "../hooks/useTracks";
import { useProjects } from "../hooks/useProjects";
import { useAgents } from "../hooks/useAgents";
import { useBoard } from "../hooks/useBoard";
import { useTrackRelations } from "../hooks/useTrackRelations";
import { useOriginSync } from "../hooks/useOriginSync";
import { useSwarm } from "../hooks/useSwarm";
import { queryKeys } from "../api/queryKeys";
import { fetcher, FetchError } from "../api/fetcher";
import { SwarmPanel } from "../components/SwarmPanel";
import { TrackList } from "../components/TrackList";
import { KanbanBoard } from "../components/KanbanBoard";
import { SyncPanel } from "../components/SyncPanel";
import { AgentTerminal } from "../components/AgentTerminal";
import { MiniCard } from "../components/MiniCard";
import { AdminPanel } from "../components/AdminPanel";
import { ProjectMetadataView } from "../components/ProjectMetadataView";
import { ConsentDialog } from "../components/ConsentDialog";
import { SkillsInstallDialog } from "../components/SkillsInstallDialog";
import { SetupRequiredDialog } from "../components/SetupRequiredDialog";
import { AgentLauncher } from "../components/AgentLauncher";
import { PaginatedList } from "../components/PaginatedList";
import { InlineSpinner } from "../components/InlineSpinner";
import { useConsent } from "../hooks/useConsent";
import { useSkillsPrompt } from "../hooks/useSkillsPrompt";
import { useSetupPrompt } from "../hooks/useSetupPrompt";
import { useProjectMetadata } from "../hooks/useProjectMetadata";
import { useProjectSettings } from "../hooks/useProjectSettings";
import { ProjectSettingsPanel } from "../components/ProjectSettingsPanel";
import appStyles from "../App.module.css";
import styles from "./ProjectPage.module.css";

function getDisabledReason(skillsMissing: boolean, setupIncomplete: boolean): string | undefined {
  if (skillsMissing) return "Install skills first";
  if (setupIncomplete) return "Run kiloforge setup first";
  return undefined;
}

interface TerminalOverlayProps {
  agentId: string;
  agents: Agent[];
  terminalKey: string;
  minimized: boolean;
  onMinimize: () => void;
  onRestore: () => void;
  onClose: () => void;
}

function TerminalOverlay({ agentId, agents, terminalKey, minimized, onMinimize, onRestore, onClose }: TerminalOverlayProps) {
  const agent = agents.find((a) => a.id === agentId);
  return (
    <>
      <AgentTerminal agentId={agentId} name={agent?.name} role={agent?.role} minimized={minimized} onMinimize={onMinimize} onClose={onClose} />
      {minimized && (
        <MiniCard agentId={agentId} name={agent?.name ?? (terminalKey === "resolver" ? "Conflict Resolver" : undefined)} role={agent?.role ?? (terminalKey === "resolver" ? "resolver" : undefined)} unreadCount={0} notificationType={null} initialX={Math.max(8, (window.innerWidth - 200) / 2)} initialY={window.innerHeight - 64} onRestore={onRestore} onClose={onClose} />
      )}
    </>
  );
}

interface SetupBannersProps {
  skillsMissing: boolean;
  setupIncomplete: boolean;
  slug: string | undefined;
  skillsPrompt: ReturnType<typeof useSkillsPrompt>;
  setupPrompt: ReturnType<typeof useSetupPrompt>;
  queryClient: ReturnType<typeof useQueryClient>;
}

function SetupBanners({ skillsMissing, setupIncomplete, slug, skillsPrompt, setupPrompt, queryClient }: SetupBannersProps) {
  return (
    <>
      {skillsMissing && (
        <div className={styles.setupBanner}>
          <span className={styles.setupBannerText}>
            Required skills not installed — install skills before running setup or spawning agents.
          </span>
          <button
            className={styles.setupBannerBtn}
            onClick={() => skillsPrompt.requestInstall(() => {
              queryClient.invalidateQueries({ queryKey: queryKeys.preflight });
            })}
            disabled={skillsPrompt.updating}
          >
            Install Skills
          </button>
        </div>
      )}
      {!skillsMissing && setupIncomplete && slug && (
        <div className={styles.setupBanner}>
          <span className={styles.setupBannerText}>
            Kiloforge setup required — run setup to configure this project for track management.
          </span>
          <button
            className={styles.setupBannerBtn}
            onClick={() => setupPrompt.requestSetup(slug, () => {
              queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus(slug) });
            })}
            disabled={setupPrompt.starting}
          >
            Run Setup
          </button>
        </div>
      )}
    </>
  );
}

function ProjectMetaSection({ project }: { project: Project }) {
  return (
    <section className={appStyles.panel}>
      <h2 className={appStyles.panelTitle}>Project</h2>
      <div className={styles.meta}>
        <div className={styles.metaRow}>
          <span className={styles.metaLabel}>Slug</span>
          <span>{project.slug}</span>
        </div>
        <div className={styles.metaRow}>
          <span className={styles.metaLabel}>Repo</span>
          <span>{project.repo_name}</span>
        </div>
        {project.origin_remote && (
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Remote</span>
            <span className={styles.mono}>{project.origin_remote}</span>
          </div>
        )}
        {project.mirror_dir && (
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Mirror</span>
            <span className={styles.mono}>{project.mirror_dir}</span>
          </div>
        )}
        <div className={styles.metaRow}>
          <span className={styles.metaLabel}>Status</span>
          <span>{project.active ? "Active" : "Inactive"}</span>
        </div>
      </div>
    </section>
  );
}

function InfoTabContent({ metadata, metadataLoading, metadataError }: {
  metadata: ProjectMetadata | undefined;
  metadataLoading: boolean;
  metadataError: Error | null;
}) {
  return (
    <section className={appStyles.panel}>
      {metadataLoading && (
        <p className={styles.metadataLoading}>Loading project metadata...</p>
      )}
      {metadataError && (
        <p className={metadataError instanceof FetchError && metadataError.status === 404 ? styles.notInitialized : styles.metadataError}>
          {metadataError instanceof FetchError && metadataError.status === 404
            ? "Kiloforge is not initialized for this project. Run setup to configure track management."
            : "Failed to load project metadata."}
        </p>
      )}
      {metadata && <ProjectMetadataView metadata={metadata} />}
    </section>
  );
}

function BoardTabPanels({ project, slug, syncStatus, syncLoading, pushing, pulling, syncError, syncConflict, onPush, onPull, onRefreshSync, onClearSyncError, onResolveConflict, swarm, swarmLoading, swarmStarting, swarmStopping, swarmUpdatingSettings, onSwarmStart, onSwarmStop, onSwarmUpdateSettings, board, boardLoading, onMoveCard, onSyncBoard, syncing, actionsDisabled, disabledReason, onDeleteTrack, dependencies, conflicts, onOpenLauncher, adminAgentId, onStartAdminOp, onSetupRequired, onSkillsRequired }: {
  project: Project | undefined;
  slug: string | undefined;
  syncStatus: ReturnType<typeof useOriginSync>["syncStatus"];
  syncLoading: boolean;
  pushing: boolean;
  pulling: boolean;
  syncError: ReturnType<typeof useOriginSync>["error"];
  syncConflict: ReturnType<typeof useOriginSync>["conflict"];
  onPush: (remoteBranch: string) => void;
  onPull: (remoteBranch?: string) => void;
  onRefreshSync: () => void;
  onClearSyncError: () => void;
  onResolveConflict: () => void;
  swarm: ReturnType<typeof useSwarm>["swarm"];
  swarmLoading: boolean;
  swarmStarting: boolean;
  swarmStopping: boolean;
  swarmUpdatingSettings: boolean;
  onSwarmStart: () => void;
  onSwarmStop: () => void;
  onSwarmUpdateSettings: ReturnType<typeof useSwarm>["updateSettings"];
  board: ReturnType<typeof useBoard>["board"];
  boardLoading: boolean;
  onMoveCard: ReturnType<typeof useBoard>["moveCard"];
  onSyncBoard: () => void;
  syncing: boolean;
  actionsDisabled: boolean;
  disabledReason: string | undefined;
  onDeleteTrack: (trackId: string) => void;
  dependencies: ReturnType<typeof useTrackRelations>["dependencies"];
  conflicts: ReturnType<typeof useTrackRelations>["conflicts"];
  onOpenLauncher: () => void;
  adminAgentId: string | null;
  onStartAdminOp: (agentId: string) => void;
  onSetupRequired: () => void;
  onSkillsRequired: () => void;
}) {
  return (
    <>
      {project?.origin_remote && (
        <section className={appStyles.panel}>
          <h2 className={appStyles.panelTitle}>Origin Sync</h2>
          <SyncPanel
            syncStatus={syncStatus}
            loading={syncLoading}
            pushing={pushing}
            pulling={pulling}
            error={syncError}
            conflict={syncConflict}
            onPush={onPush}
            onPull={onPull}
            onRefresh={onRefreshSync}
            onClearError={onClearSyncError}
            onResolveConflict={onResolveConflict}
          />
        </section>
      )}

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>AI Agent Swarm</h2>
        <SwarmPanel
          swarm={swarm}
          loading={swarmLoading}
          starting={swarmStarting}
          stopping={swarmStopping}
          updatingSettings={swarmUpdatingSettings}
          onStart={onSwarmStart}
          onStop={onSwarmStop}
          onUpdateSettings={onSwarmUpdateSettings}
        />
      </section>

      <section className={appStyles.panel} data-tour="board-section">
        <div className={styles.boardHeader}>
          <h2 className={appStyles.panelTitle}>Board</h2>
          <div className={styles.boardActions}>
            <button className={styles.syncBtn} onClick={onSyncBoard} disabled={syncing || actionsDisabled} title={disabledReason}>
              {syncing ? "Syncing..." : "Sync"}
            </button>
            <button className={styles.generateBtn} onClick={onOpenLauncher} disabled={actionsDisabled} title={disabledReason} data-tour="generate-tracks">
              New Agent
            </button>
          </div>
        </div>
        {boardLoading ? (
          <InlineSpinner label="Loading board..." />
        ) : (
          <KanbanBoard
            board={board ?? { columns: ["backlog", "approved", "in_progress", "done"], cards: {} }}
            projectSlug={slug}
            onMoveCard={onMoveCard}
            onDeleteTrack={onDeleteTrack}
            dependencies={dependencies}
            conflicts={conflicts}
          />
        )}
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Admin Operations</h2>
        <AdminPanel
          projectSlug={slug}
          running={adminAgentId !== null}
          disabled={actionsDisabled}
          disabledReason={disabledReason}
          onStartOperation={onStartAdminOp}
          onSetupRequired={onSetupRequired}
          onSkillsRequired={onSkillsRequired}
        />
      </section>
    </>
  );
}

function TrackSearchSection({ tracks, tracksLoading, trackRemaining, trackHasNext, trackFetching, trackLoadMore, slug }: {
  tracks: ReturnType<typeof useTracks>["tracks"];
  tracksLoading: boolean;
  trackRemaining: number;
  trackHasNext: boolean;
  trackFetching: boolean;
  trackLoadMore: () => void;
  slug: string | undefined;
}) {
  const [trackSearch, setTrackSearch] = useState("");
  const filteredTracks = trackSearch
    ? tracks.filter((t) => t.title.toLowerCase().includes(trackSearch.toLowerCase()) || t.id.toLowerCase().includes(trackSearch.toLowerCase()))
    : tracks;

  return (
    <section className={appStyles.panel}>
      <h2 className={appStyles.panelTitle}>Tracks</h2>
      {tracksLoading ? (
        <InlineSpinner label="Loading tracks..." />
      ) : (
        <>
          <div className={styles.trackSearchWrap}>
            <input type="text" className={styles.trackSearchInput} placeholder="Search tracks..." value={trackSearch} onChange={(e) => setTrackSearch(e.target.value)} />
            {trackSearch && (
              <button className={styles.trackSearchClear} onClick={() => setTrackSearch("")} aria-label="Clear search">&times;</button>
            )}
          </div>
          <PaginatedList remainingCount={trackRemaining} hasNextPage={trackHasNext} isFetchingNextPage={trackFetching} onLoadMore={trackLoadMore}>
            <TrackList tracks={filteredTracks} projectSlug={slug} />
          </PaginatedList>
        </>
      )}
    </section>
  );
}

function ProjectDialogs({ showLauncher, onLaunch, onCloseLauncher, launchPending, slug, consent, skillsPrompt, setupPrompt, agents, onSetupComplete }: {
  showLauncher: boolean;
  onLaunch: (role: AgentRole, prompt: string) => void;
  onCloseLauncher: () => void;
  launchPending: boolean;
  slug: string | undefined;
  consent: ReturnType<typeof useConsent>;
  skillsPrompt: ReturnType<typeof useSkillsPrompt>;
  setupPrompt: ReturnType<typeof useSetupPrompt>;
  agents: Agent[];
  onSetupComplete: () => void;
}) {
  const setupAgent = setupPrompt.agentId ? agents.find((a) => a.id === setupPrompt.agentId) : undefined;
  return (
    <>
      {showLauncher && (
        <AgentLauncher
          onLaunch={onLaunch}
          onClose={onCloseLauncher}
          launching={launchPending}
          projectSlug={slug}
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
      {setupPrompt.showDialog && (
        <SetupRequiredDialog
          projectSlug={setupPrompt.projectSlug}
          agentId={setupPrompt.agentId}
          agentName={setupAgent?.name}
          agentRole={setupAgent?.role}
          starting={setupPrompt.starting}
          error={setupPrompt.error}
          onRunSetup={setupPrompt.startSetup}
          onSetupComplete={onSetupComplete}
          onCancel={setupPrompt.cancel}
        />
      )}
    </>
  );
}

export function ProjectPage() {
  const { slug } = useParams<{ slug: string }>();
  const { tracks, loading: tracksLoading, remainingCount: trackRemaining, hasNextPage: trackHasNext, isFetchingNextPage: trackFetching, fetchNextPage: trackLoadMore } = useTracks(slug);
  const { projects } = useProjects();
  const { board, loading: boardLoading, moveCard, syncBoard, syncing } = useBoard(slug);
  const boardTrackIds = useMemo(() => Object.keys(board?.cards ?? {}), [board]);
  const { dependencies, conflicts } = useTrackRelations(boardTrackIds, slug);
  const { syncStatus, loading: syncLoading, pushing, pulling, error: syncError, conflict: syncConflict, push, pull, refresh: refreshSync, clearError: clearSyncError, clearConflict: clearSyncConflict } = useOriginSync(slug);
  const { swarm, loading: swarmLoading, starting: swarmStarting, stopping: swarmStopping, updatingSettings: swarmUpdatingSettings, start: swarmStart, stop: swarmStop, updateSettings: swarmUpdateSettings } = useSwarm(slug);
  const project = projects.find((p) => p.slug === slug);
  const { agents } = useAgents();

  const queryClient = useQueryClient();
  const [showLauncher, setShowLauncher] = useState(false);
  const [terminalAgentId, setTerminalAgentId] = useState<string | null>(null);
  const [adminAgentId, setAdminAgentId] = useState<string | null>(null);
  const { data: setupStatus } = useQuery({
    queryKey: queryKeys.setupStatus(slug ?? ""),
    queryFn: () =>
      fetcher<{ setup_complete: boolean; project_slug: string }>(
        `/api/projects/${encodeURIComponent(slug!)}/setup-status`,
      ),
    enabled: !!slug,
  });
  const { data: preflight } = useQuery({
    queryKey: queryKeys.preflight,
    queryFn: () =>
      fetcher<{
        claude_authenticated: boolean;
        skills_ok: boolean;
        skills_missing?: string[];
        consent_given: boolean;
        setup_required: boolean;
      }>("/api/preflight"),
  });

  const skillsMissing = preflight !== undefined && !preflight.skills_ok;
  const setupIncomplete = !skillsMissing && setupStatus !== undefined && !setupStatus.setup_complete;
  const actionsDisabled = skillsMissing || setupIncomplete;
  const disabledReason = getDisabledReason(skillsMissing, setupIncomplete);

  const [pageTab, setPageTab] = useState<"board" | "info" | "settings">("board");
  const { settings: projectSettings, loading: settingsLoading, updating: settingsUpdating, updateSettings } = useProjectSettings(slug);
  const { data: metadata, isLoading: metadataLoading, error: metadataError } = useProjectMetadata(slug);
  const consent = useConsent();
  const skillsPrompt = useSkillsPrompt();
  const setupPrompt = useSetupPrompt({
    onConsentRequired: (retry) => consent.requestConsent(retry),
  });
  const handleSetupComplete = useCallback(() => {
    if (slug) {
      queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus(slug) });
    }
    setupPrompt.handleSetupComplete();
  }, [slug, queryClient, setupPrompt]);

  const handlePush = useCallback((remoteBranch: string) => {
    push({ remote_branch: remoteBranch });
  }, [push]);

  const handlePull = useCallback((remoteBranch?: string) => {
    pull(remoteBranch);
  }, [pull]);

  const [resolverAgentId, setResolverAgentId] = useState<string | null>(null);
  const [minimizedTerminals, setMinimizedTerminals] = useState<Set<string>>(new Set());
  const [lastResolveReq, setLastResolveReq] = useState<ResolveConflictRequest | null>(null);

  const resolveConflictMutation = useMutation({
    mutationFn: (req: ResolveConflictRequest) =>
      fetcher<Agent>(`/api/projects/${encodeURIComponent(slug!)}/resolve-conflict`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }),
    onSuccess: (agent) => {
      setResolverAgentId(agent.id);
    },
    onError: (err) => {
      if (err instanceof FetchError && err.status === 412) {
        skillsPrompt.requestInstall(() => {
          if (lastResolveReq) resolveConflictMutation.mutate(lastResolveReq);
        });
      }
    },
  });

  const handleResolveConflict = useCallback(() => {
    if (!syncConflict || !slug) return;
    const req: ResolveConflictRequest = {
      direction: syncConflict.direction,
      remote_branch: "kf/main",
    };
    setLastResolveReq(req);
    resolveConflictMutation.mutate(req);
  }, [syncConflict, slug, resolveConflictMutation]);

  const handleResolverTerminalClose = useCallback(() => {
    setResolverAgentId(null);
    setMinimizedTerminals((prev) => { const next = new Set(prev); next.delete("resolver"); return next; });
    clearSyncConflict();
    if (slug) {
      queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(slug) });
    }
  }, [clearSyncConflict, queryClient, slug]);

  const [lastSpawnReq, setLastSpawnReq] = useState<SpawnInteractiveRequest>({});

  const spawnMutation = useMutation({
    mutationFn: (req: SpawnInteractiveRequest) =>
      fetcher<Agent>("/api/agents/interactive", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }),
    onSuccess: (agent) => {
      setTerminalAgentId(agent.id);
      setShowLauncher(false);
    },
    onError: (err) => {
      if (err instanceof FetchError && err.status === 403) {
        consent.requestConsent(() => spawnMutation.mutate(lastSpawnReq));
      } else if (err instanceof FetchError && err.status === 412) {
        skillsPrompt.requestInstall(() => spawnMutation.mutate(lastSpawnReq));
      } else if (err instanceof FetchError && err.status === 428 && slug) {
        setupPrompt.requestSetup(slug, () => spawnMutation.mutate(lastSpawnReq));
      }
    },
  });

  const handleLaunch = useCallback((role: AgentRole, prompt: string) => {
    const req: SpawnInteractiveRequest = { role, project: slug };
    if (prompt) req.prompt = prompt;
    setLastSpawnReq(req);
    spawnMutation.mutate(req);
  }, [slug, spawnMutation]);

  const deleteMutation = useMutation({
    mutationFn: (trackId: string) =>
      fetcher<void>(
        `/api/tracks/${encodeURIComponent(trackId)}?project=${encodeURIComponent(slug!)}`,
        { method: "DELETE" },
      ),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.board(slug ?? "") });
    },
  });

  const handleDeleteTrack = useCallback(
    (trackId: string) => {
      if (!slug) return;
      deleteMutation.mutate(trackId);
    },
    [slug, deleteMutation],
  );

  const handleTerminalClose = useCallback(() => {
    setTerminalAgentId(null);
    setMinimizedTerminals((prev) => { const next = new Set(prev); next.delete("terminal"); return next; });
    queryClient.invalidateQueries({ queryKey: queryKeys.board(slug ?? "") });
  }, [queryClient, slug]);

  const handleAdminTerminalClose = useCallback(() => {
    setAdminAgentId(null);
    setMinimizedTerminals((prev) => { const next = new Set(prev); next.delete("admin"); return next; });
    queryClient.invalidateQueries({ queryKey: queryKeys.board(slug ?? "") });
  }, [queryClient, slug]);

  const minimizeTerminal = useCallback((key: string) => {
    setMinimizedTerminals((prev) => new Set(prev).add(key));
  }, []);

  const restoreTerminal = useCallback((key: string) => {
    setMinimizedTerminals((prev) => { const next = new Set(prev); next.delete(key); return next; });
  }, []);

  return (
    <>
      <div className={styles.breadcrumb}>
        <Link to="/" className={styles.backLink}>Overview</Link>
        <span className={styles.separator}>/</span>
        <span>{slug}</span>
      </div>

      {project && <ProjectMetaSection project={project} />}

      {/* Page-level tabs */}
      <div className={styles.pageTabs}>
        <button
          className={`${styles.pageTab} ${pageTab === "board" ? styles.pageTabActive : ""}`}
          onClick={() => setPageTab("board")}
        >
          Board
        </button>
        <button
          className={`${styles.pageTab} ${pageTab === "info" ? styles.pageTabActive : ""}`}
          onClick={() => setPageTab("info")}
        >
          Project Info
        </button>
        <button
          className={`${styles.pageTab} ${pageTab === "settings" ? styles.pageTabActive : ""}`}
          onClick={() => setPageTab("settings")}
        >
          Settings
        </button>
      </div>

      {pageTab === "info" && <InfoTabContent metadata={metadata} metadataLoading={metadataLoading} metadataError={metadataError} />}

      {pageTab === "settings" && (
        <section className={appStyles.panel}>
          <h2 className={appStyles.panelTitle}>Project Settings</h2>
          <ProjectSettingsPanel
            settings={projectSettings}
            loading={settingsLoading}
            updating={settingsUpdating}
            onUpdate={updateSettings}
          />
        </section>
      )}

      {pageTab === "board" && (<>
      <SetupBanners
        skillsMissing={skillsMissing}
        setupIncomplete={setupIncomplete}
        slug={slug}
        skillsPrompt={skillsPrompt}
        setupPrompt={setupPrompt}
        queryClient={queryClient}
      />

      <BoardTabPanels
        project={project}
        slug={slug}
        syncStatus={syncStatus}
        syncLoading={syncLoading}
        pushing={pushing}
        pulling={pulling}
        syncError={syncError}
        syncConflict={syncConflict}
        onPush={handlePush}
        onPull={handlePull}
        onRefreshSync={refreshSync}
        onClearSyncError={clearSyncError}
        onResolveConflict={handleResolveConflict}
        swarm={swarm}
        swarmLoading={swarmLoading}
        swarmStarting={swarmStarting}
        swarmStopping={swarmStopping}
        swarmUpdatingSettings={swarmUpdatingSettings}
        onSwarmStart={swarmStart}
        onSwarmStop={swarmStop}
        onSwarmUpdateSettings={swarmUpdateSettings}
        board={board}
        boardLoading={boardLoading}
        onMoveCard={moveCard}
        onSyncBoard={syncBoard}
        syncing={syncing}
        actionsDisabled={actionsDisabled}
        disabledReason={disabledReason}
        onDeleteTrack={handleDeleteTrack}
        dependencies={dependencies}
        conflicts={conflicts}
        onOpenLauncher={() => setShowLauncher(true)}
        adminAgentId={adminAgentId}
        onStartAdminOp={setAdminAgentId}
        onSetupRequired={() => {
          if (slug) setupPrompt.requestSetup(slug, () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus(slug) });
          });
        }}
        onSkillsRequired={() => {
          skillsPrompt.requestInstall(() => {
            queryClient.invalidateQueries({ queryKey: queryKeys.preflight });
          });
        }}
      />

      {resolverAgentId && (
        <TerminalOverlay agentId={resolverAgentId} agents={agents} terminalKey="resolver" minimized={minimizedTerminals.has("resolver")} onMinimize={() => minimizeTerminal("resolver")} onRestore={() => restoreTerminal("resolver")} onClose={handleResolverTerminalClose} />
      )}
      {terminalAgentId && (
        <TerminalOverlay agentId={terminalAgentId} agents={agents} terminalKey="terminal" minimized={minimizedTerminals.has("terminal")} onMinimize={() => minimizeTerminal("terminal")} onRestore={() => restoreTerminal("terminal")} onClose={handleTerminalClose} />
      )}
      {adminAgentId && (
        <TerminalOverlay agentId={adminAgentId} agents={agents} terminalKey="admin" minimized={minimizedTerminals.has("admin")} onMinimize={() => minimizeTerminal("admin")} onRestore={() => restoreTerminal("admin")} onClose={handleAdminTerminalClose} />
      )}

      <ProjectDialogs
        showLauncher={showLauncher}
        onLaunch={handleLaunch}
        onCloseLauncher={() => setShowLauncher(false)}
        launchPending={spawnMutation.isPending}
        slug={slug}
        consent={consent}
        skillsPrompt={skillsPrompt}
        setupPrompt={setupPrompt}
        agents={agents}
        onSetupComplete={handleSetupComplete}
      />

      <TrackSearchSection
        tracks={tracks}
        tracksLoading={tracksLoading}
        trackRemaining={trackRemaining}
        trackHasNext={trackHasNext}
        trackFetching={trackFetching}
        trackLoadMore={trackLoadMore}
        slug={slug}
      />
      </>)}
    </>
  );
}
