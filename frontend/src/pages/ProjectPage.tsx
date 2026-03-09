import { useCallback, useState, useEffect } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTracks } from "../hooks/useTracks";
import { useProjects } from "../hooks/useProjects";
import { useBoard } from "../hooks/useBoard";
import { useOriginSync } from "../hooks/useOriginSync";
import { queryKeys } from "../api/queryKeys";
import { fetcher, FetchError } from "../api/fetcher";
import { TrackList } from "../components/TrackList";
import { KanbanBoard } from "../components/KanbanBoard";
import { SyncPanel } from "../components/SyncPanel";
import { AgentTerminal } from "../components/AgentTerminal";
import { AdminPanel } from "../components/AdminPanel";
import { ProjectMetadataView } from "../components/ProjectMetadataView";
import { ConsentDialog } from "../components/ConsentDialog";
import { SkillsInstallDialog } from "../components/SkillsInstallDialog";
import { SetupRequiredDialog } from "../components/SetupRequiredDialog";
import { useConsent } from "../hooks/useConsent";
import { useSkillsPrompt } from "../hooks/useSkillsPrompt";
import { useSetupPrompt } from "../hooks/useSetupPrompt";
import { useProjectMetadata } from "../hooks/useProjectMetadata";
import { useTourContextSafe } from "../components/tour/TourProvider";
import { TOUR_STEPS } from "../components/tour/tourSteps";
import appStyles from "../App.module.css";
import styles from "./ProjectPage.module.css";

export function ProjectPage() {
  const { slug } = useParams<{ slug: string }>();
  const { tracks, loading: tracksLoading } = useTracks(slug);
  const { projects } = useProjects();
  const { board, loading: boardLoading, moveCard, syncBoard, syncing } = useBoard(slug);
  const { syncStatus, loading: syncLoading, pushing, pulling, error: syncError, push, pull, refresh: refreshSync, clearError: clearSyncError } = useOriginSync(slug);
  const project = projects.find((p) => p.slug === slug);

  const queryClient = useQueryClient();
  const [showPrompt, setShowPrompt] = useState(false);
  const [prompt, setPrompt] = useState("");
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

  const [pageTab, setPageTab] = useState<"board" | "info">("board");
  const { data: metadata, isLoading: metadataLoading, error: metadataError } = useProjectMetadata(slug);
  const consent = useConsent();
  const skillsPrompt = useSkillsPrompt();
  const setupPrompt = useSetupPrompt();
  const tour = useTourContextSafe();

  const handleSetupComplete = useCallback(() => {
    if (slug) {
      queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus(slug) });
    }
    setupPrompt.handleSetupComplete();
  }, [slug, queryClient, setupPrompt]);

  // Tour: auto-show prompt and prefill when on generate-tracks step
  const tourStep = tour?.isActive ? TOUR_STEPS[tour.currentStep] : null;
  useEffect(() => {
    if (tourStep?.id === "generate-tracks" && !showPrompt) {
      setShowPrompt(true);
      setPrompt("Add user authentication with login, registration, and password reset");
    }
  }, [tourStep?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const handlePush = useCallback((remoteBranch: string) => {
    push({ remote_branch: remoteBranch });
  }, [push]);

  const handlePull = useCallback((remoteBranch?: string) => {
    pull(remoteBranch);
  }, [pull]);

  const generateMutation = useMutation({
    mutationFn: (p: string) =>
      fetcher<{ agent_id: string; ws_url: string }>("/api/tracks/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt: p, project: slug }),
      }),
    onSuccess: (data) => {
      setTerminalAgentId(data.agent_id);
      setShowPrompt(false);
      setPrompt("");
    },
    onError: (err) => {
      if (err instanceof FetchError && err.status === 403) {
        consent.requestConsent(() => handleGenerateTracks());
      } else if (err instanceof FetchError && err.status === 412) {
        skillsPrompt.requestInstall(() => handleGenerateTracks());
      } else if (err instanceof FetchError && err.status === 428 && slug) {
        setupPrompt.requestSetup(slug, () => handleGenerateTracks());
      }
    },
  });

  const handleGenerateTracks = useCallback(() => {
    if (!prompt.trim() || !slug) return;
    generateMutation.mutate(prompt.trim());
  }, [prompt, slug, generateMutation, consent]);

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
              onClick={() => { if (!actionsDisabled) setShowPrompt((v) => !v); }}
              disabled={actionsDisabled}
              title={disabledReason}
              data-tour="generate-tracks"
            >
              Generate Tracks
            </button>
          </div>
        </div>
        {showPrompt && (
          <div className={styles.promptForm}>
            <textarea
              className={styles.promptInput}
              placeholder="Describe the features or changes you want to generate tracks for..."
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              rows={3}
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  handleGenerateTracks();
                }
              }}
            />
            <div className={styles.promptActions}>
              <button
                className={styles.promptSubmit}
                disabled={!prompt.trim() || generateMutation.isPending}
                onClick={handleGenerateTracks}
              >
                {generateMutation.isPending ? "Starting..." : "Generate"}
              </button>
              <button className={styles.promptCancel} onClick={() => { setShowPrompt(false); setPrompt(""); }}>
                Cancel
              </button>
            </div>
          </div>
        )}
        {boardLoading ? (
          <p className={appStyles.empty}>Loading board...</p>
        ) : (
          <KanbanBoard
            board={board ?? { columns: ["backlog", "approved", "in_progress", "in_review", "done"], cards: {} }}
            projectSlug={slug}
            onMoveCard={moveCard}
            onDeleteTrack={handleDeleteTrack}
          />
        )}
      </section>

      {terminalAgentId && (
        <AgentTerminal agentId={terminalAgentId} onClose={handleTerminalClose} />
      )}

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

      {adminAgentId && (
        <AgentTerminal agentId={adminAgentId} onClose={handleAdminTerminalClose} />
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
