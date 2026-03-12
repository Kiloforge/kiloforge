import { useCallback, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Agent, ResolveConflictRequest, SpawnInteractiveRequest } from "../types/api";
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

export function ProjectPage() {
  const { slug } = useParams<{ slug: string }>();
  const { tracks, loading: tracksLoading, remainingCount: trackRemaining, hasNextPage: trackHasNext, isFetchingNextPage: trackFetching, fetchNextPage: trackLoadMore } = useTracks(slug);
  const { projects } = useProjects();
  const { board, loading: boardLoading, moveCard, syncBoard, syncing } = useBoard(slug);
  const boardTrackIds = useMemo(() => board ? Object.keys(board.cards) : [], [board]);
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
  const disabledReason = skillsMissing
    ? "Install skills first"
    : setupIncomplete
    ? "Run kiloforge setup first"
    : undefined;

  const [pageTab, setPageTab] = useState<"board" | "info" | "settings">("board");
  const [trackSearch, setTrackSearch] = useState("");
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

      {project && (
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
      )}

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

      {pageTab === "info" && (
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
      )}

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
      {!skillsMissing && setupStatus && !setupStatus.setup_complete && slug && (
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
            onPush={handlePush}
            onPull={handlePull}
            onRefresh={refreshSync}
            onClearError={clearSyncError}
            onResolveConflict={handleResolveConflict}
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
          onStart={() => swarmStart()}
          onStop={swarmStop}
          onUpdateSettings={swarmUpdateSettings}
        />
      </section>

      <section className={appStyles.panel} data-tour="board-section">
        <div className={styles.boardHeader}>
          <h2 className={appStyles.panelTitle}>Board</h2>
          <div className={styles.boardActions}>
            <button
              className={styles.syncBtn}
              onClick={syncBoard}
              disabled={syncing || actionsDisabled}
              title={disabledReason}
            >
              {syncing ? "Syncing..." : "Sync"}
            </button>
            <button
              className={styles.generateBtn}
              onClick={() => { if (!actionsDisabled) setShowLauncher(true); }}
              disabled={actionsDisabled}
              title={disabledReason}
              data-tour="generate-tracks"
            >
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
            onMoveCard={moveCard}
            onDeleteTrack={handleDeleteTrack}
            dependencies={dependencies}
            conflicts={conflicts}
          />
        )}
      </section>

      {resolverAgentId && (() => {
        const agent = agents.find((a) => a.id === resolverAgentId);
        return (
          <>
            <AgentTerminal agentId={resolverAgentId} name={agent?.name ?? "Conflict Resolver"} role={agent?.role ?? "resolver"} minimized={minimizedTerminals.has("resolver")} onMinimize={() => minimizeTerminal("resolver")} onClose={handleResolverTerminalClose} />
            {minimizedTerminals.has("resolver") && (
              <MiniCard agentId={resolverAgentId} name={agent?.name ?? "Conflict Resolver"} role={agent?.role ?? "resolver"} unreadCount={0} notificationType={null} initialX={Math.max(8, (window.innerWidth - 200) / 2)} initialY={window.innerHeight - 64} onRestore={() => restoreTerminal("resolver")} onClose={handleResolverTerminalClose} />
            )}
          </>
        );
      })()}

      {terminalAgentId && (() => {
        const agent = agents.find((a) => a.id === terminalAgentId);
        return (
          <>
            <AgentTerminal agentId={terminalAgentId} name={agent?.name} role={agent?.role} minimized={minimizedTerminals.has("terminal")} onMinimize={() => minimizeTerminal("terminal")} onClose={handleTerminalClose} />
            {minimizedTerminals.has("terminal") && (
              <MiniCard agentId={terminalAgentId} name={agent?.name} role={agent?.role} unreadCount={0} notificationType={null} initialX={Math.max(8, (window.innerWidth - 200) / 2)} initialY={window.innerHeight - 64} onRestore={() => restoreTerminal("terminal")} onClose={handleTerminalClose} />
            )}
          </>
        );
      })()}

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Admin Operations</h2>
        <AdminPanel
          projectSlug={slug}
          running={adminAgentId !== null}
          disabled={actionsDisabled}
          disabledReason={disabledReason}
          onStartOperation={setAdminAgentId}
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
      </section>

      {adminAgentId && (() => {
        const agent = agents.find((a) => a.id === adminAgentId);
        return (
          <>
            <AgentTerminal agentId={adminAgentId} name={agent?.name} role={agent?.role} minimized={minimizedTerminals.has("admin")} onMinimize={() => minimizeTerminal("admin")} onClose={handleAdminTerminalClose} />
            {minimizedTerminals.has("admin") && (
              <MiniCard agentId={adminAgentId} name={agent?.name} role={agent?.role} unreadCount={0} notificationType={null} initialX={Math.max(8, (window.innerWidth - 200) / 2)} initialY={window.innerHeight - 64} onRestore={() => restoreTerminal("admin")} onClose={handleAdminTerminalClose} />
            )}
          </>
        );
      })()}

      {showLauncher && (
        <AgentLauncher
          onLaunch={handleLaunch}
          onClose={() => setShowLauncher(false)}
          launching={spawnMutation.isPending}
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
      {setupPrompt.showDialog && (() => {
        const setupAgent = setupPrompt.agentId ? agents.find((a) => a.id === setupPrompt.agentId) : undefined;
        return (
          <SetupRequiredDialog
            projectSlug={setupPrompt.projectSlug}
            agentId={setupPrompt.agentId}
            agentName={setupAgent?.name}
            agentRole={setupAgent?.role}
            starting={setupPrompt.starting}
            error={setupPrompt.error}
            onRunSetup={setupPrompt.startSetup}
            onSetupComplete={handleSetupComplete}
            onCancel={setupPrompt.cancel}
          />
        );
      })()}

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Tracks</h2>
        {tracksLoading ? (
          <InlineSpinner label="Loading tracks..." />
        ) : (
          <>
            <div className={styles.trackSearchWrap}>
              <input
                type="text"
                className={styles.trackSearchInput}
                placeholder="Search tracks..."
                value={trackSearch}
                onChange={(e) => setTrackSearch(e.target.value)}
              />
              {trackSearch && (
                <button
                  className={styles.trackSearchClear}
                  onClick={() => setTrackSearch("")}
                  aria-label="Clear search"
                >
                  &times;
                </button>
              )}
            </div>
            <PaginatedList
              remainingCount={trackRemaining}
              hasNextPage={trackHasNext}
              isFetchingNextPage={trackFetching}
              onLoadMore={() => trackLoadMore()}
            >
              <TrackList
                tracks={trackSearch
                  ? tracks.filter((t) => t.title.toLowerCase().includes(trackSearch.toLowerCase()) || t.id.toLowerCase().includes(trackSearch.toLowerCase()))
                  : tracks}
                projectSlug={slug}
              />
            </PaginatedList>
          </>
        )}
      </section>
      </>)}
    </>
  );
}
