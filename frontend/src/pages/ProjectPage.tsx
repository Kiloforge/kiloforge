import { useCallback } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTracks } from "../hooks/useTracks";
import { useProjects } from "../hooks/useProjects";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { TrackList } from "../components/TrackList";
import { ConsentDialog } from "../components/ConsentDialog";
import { SkillsInstallDialog } from "../components/SkillsInstallDialog";
import { SetupRequiredDialog } from "../components/SetupRequiredDialog";
import { useConsent } from "../hooks/useConsent";
import { useSkillsPrompt } from "../hooks/useSkillsPrompt";
import { useSetupPrompt } from "../hooks/useSetupPrompt";
import { SyncContainer } from "../containers/SyncContainer";
import { BoardContainer } from "../containers/BoardContainer";
import { AdminContainer } from "../containers/AdminContainer";
import appStyles from "../App.module.css";
import styles from "./ProjectPage.module.css";

export function ProjectPage() {
  const { slug } = useParams<{ slug: string }>();
  const { tracks, loading: tracksLoading } = useTracks(slug);
  const { projects } = useProjects();
  const project = projects.find((p) => p.slug === slug);

  const queryClient = useQueryClient();
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

  const consent = useConsent();
  const skillsPrompt = useSkillsPrompt();
  const setupPrompt = useSetupPrompt();

  const handleSetupComplete = useCallback(() => {
    if (slug) {
      queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus(slug) });
    }
    setupPrompt.handleSetupComplete();
  }, [slug, queryClient, setupPrompt]);

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

      {project?.origin_remote && slug && <SyncContainer slug={slug} />}

      {slug && (
        <BoardContainer
          slug={slug}
          actionsDisabled={actionsDisabled}
          disabledReason={disabledReason}
          onConsentRequired={(retry) => consent.requestConsent(retry)}
          onSkillsRequired={(retry) => skillsPrompt.requestInstall(retry)}
          onSetupRequired={(s, retry) => setupPrompt.requestSetup(s, retry)}
        />
      )}

      {slug && (
        <AdminContainer
          slug={slug}
          actionsDisabled={actionsDisabled}
          disabledReason={disabledReason}
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
    </>
  );
}
