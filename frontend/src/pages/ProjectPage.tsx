import { useCallback, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Agent, SpawnInteractiveRequest } from "../types/api";
import type { AgentRole } from "../components/AgentLauncher";
import { useTracks } from "../hooks/useTracks";
import { useProjects } from "../hooks/useProjects";
import { useAgents } from "../hooks/useAgents";
import { useBoard } from "../hooks/useBoard";
import { useTrackRelations } from "../hooks/useTrackRelations";
import { useOriginSync } from "../hooks/useOriginSync";
import { useQueue } from "../hooks/useQueue";
import { queryKeys } from "../api/queryKeys";
import { fetcher, FetchError } from "../api/fetcher";
import { QueuePanel } from "../components/QueuePanel";
import { TrackList } from "../components/TrackList";
import { KanbanBoard } from "../components/KanbanBoard";
import { SyncPanel } from "../components/SyncPanel";
import { AgentTerminal } from "../components/AgentTerminal";
import { AdminPanel } from "../components/AdminPanel";
import { ProjectMetadataView } from "../components/ProjectMetadataView";
import { ConsentDialog } from "../components/ConsentDialog";
import { SkillsInstallDialog } from "../components/SkillsInstallDialog";
import { SetupRequiredDialog } from "../components/SetupRequiredDialog";
import { AgentLauncher } from "../components/AgentLauncher";
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
  const { tracks, loading: tracksLoading } = useTracks(slug);
  const { projects } = useProjects();
  const { board, loading: boardLoading, moveCard, syncBoard, syncing } = useBoard(slug);
  const boardTrackIds = useMemo(() => board ? Object.keys(board.cards) : [], [board]);
  const { dependencies, conflicts } = useTrackRelations(boardTrackIds, slug);
  const { syncStatus, loading: syncLoading, pushing, pulling, error: syncError, push, pull, refresh: refreshSync, clearError: clearSyncError } = useOriginSync(slug);
  const { queue, loading: queueLoading, starting: queueStarting, stopping: queueStopping, updatingSettings: queueUpdatingSettings, start: queueStart, stop: queueStop, updateSettings: queueUpdateSettings } = useQueue(slug);
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
  const { settings: projectSettings, loading: settingsLoading, updating: settingsUpdating, updateSettings } = useProjectSettings(slug);
  const { data: metadata, isLoading: metadataLoading, error: metadataError } = useProjectMetadata(slug);
  const consent = useConsent();
  const skillsPrompt = useSkillsPrompt();
  const setupPrompt = useSetupPrompt();
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
    queryClient.invalidateQueries({ queryKey: queryKeys.board(slug ?? "") });
  }, [queryClient, slug]);

  const handleAdminTerminalClose = useCallback(() => {
    setAdminAgentId(null);
    queryClient.invalidateQueries({ queryKey: queryKeys.board(slug ?? "") });
  }, [queryClient, slug]);

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
            onPush={handlePush}
            onPull={handlePull}
            onRefresh={refreshSync}
            onClearError={clearSyncError}
          />
        </section>
      )}

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Work Queue</h2>
        <QueuePanel
          queue={queue}
          loading={queueLoading}
          starting={queueStarting}
          stopping={queueStopping}
          updatingSettings={queueUpdatingSettings}
          onStart={() => queueStart()}
          onStop={queueStop}
          onUpdateSettings={queueUpdateSettings}
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
          <p className={appStyles.empty}>Loading board...</p>
        ) : (
          <KanbanBoard
            board={board ?? { columns: ["backlog", "approved", "in_progress", "in_review", "done"], cards: {} }}
            projectSlug={slug}
            onMoveCard={moveCard}
            onDeleteTrack={handleDeleteTrack}
            dependencies={dependencies}
            conflicts={conflicts}
          />
        )}
      </section>

      {terminalAgentId && (() => {
        const agent = agents.find((a) => a.id === terminalAgentId);
        return <AgentTerminal agentId={terminalAgentId} name={agent?.name} role={agent?.role} onClose={handleTerminalClose} />;
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
        return <AgentTerminal agentId={adminAgentId} name={agent?.name} role={agent?.role} onClose={handleAdminTerminalClose} />;
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
      {setupPrompt.showDialog && (
        <SetupRequiredDialog
          projectSlug={setupPrompt.projectSlug}
          agentId={setupPrompt.agentId}
          starting={setupPrompt.starting}
          error={setupPrompt.error}
          onRunSetup={setupPrompt.startSetup}
          onSetupComplete={handleSetupComplete}
          onCancel={setupPrompt.cancel}
        />
      )}

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Tracks</h2>
        {tracksLoading ? (
          <p className={appStyles.empty}>Loading tracks...</p>
        ) : (
          <TrackList tracks={tracks} projectSlug={slug} />
        )}
      </section>
      </>)}
    </>
  );
}
