import { useState, useMemo, useCallback } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import type { Agent, QuotaResponse, Track, Project, SyncStatus } from "../types/api";
import { useProjects } from "../hooks/useProjects";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { StatCards } from "../components/StatCards";
import { AgentGrid } from "../components/AgentGrid";
import { TrackList } from "../components/TrackList";
import { TraceList } from "../components/TraceList";
import { AddProjectForm } from "../components/AddProjectForm";
import { RemoveProjectDialog } from "../components/RemoveProjectDialog";
import { useTourContextSafe } from "../components/tour/TourProvider";
import { TOUR_STEPS } from "../components/tour/tourSteps";
import { useTraces } from "../hooks/useTraces";
import { useConfig } from "../hooks/useConfig";
import styles from "./OverviewPage.module.css";
import appStyles from "../App.module.css";

interface OverviewPageProps {
  agents: Agent[];
  agentsLoading: boolean;
  quota: QuotaResponse | null;
  tracks: Track[];
  onViewLog: (agentId: string) => void;
  onAttach?: (agentId: string) => void;
  onSpawnInteractive?: () => void;
  spawningInteractive?: boolean;
}

function trackCountsByStatus(tracks: Track[], slug: string) {
  const projectTracks = tracks.filter((t) => t.project === slug);
  const counts = { total: projectTracks.length, complete: 0, pending: 0, "in-progress": 0 };
  for (const t of projectTracks) {
    if (t.status === "complete") counts.complete++;
    else if (t.status === "in-progress") counts["in-progress"]++;
    else counts.pending++;
  }
  return counts;
}

function SyncBadge({ slug }: { slug: string }) {
  const { data: status = null } = useQuery({
    queryKey: queryKeys.syncStatus(slug),
    queryFn: () => fetcher<SyncStatus>(`/api/projects/${encodeURIComponent(slug)}/sync-status`).catch(() => null),
  });

  if (!status) return <span className={styles.syncBadge} title="Sync unknown">&#x2500;</span>;

  const map: Record<string, { label: string; cls: string; icon: string }> = {
    synced: { label: "Synced", cls: styles.syncSynced, icon: "\u2713" },
    ahead: { label: `${status.ahead} ahead`, cls: styles.syncAhead, icon: "\u2191" },
    behind: { label: `${status.behind} behind`, cls: styles.syncBehind, icon: "\u2193" },
    diverged: { label: `${status.ahead}\u2191 ${status.behind}\u2193`, cls: styles.syncDiverged, icon: "\u21c5" },
    unknown: { label: "Unknown", cls: "", icon: "?" },
  };
  const info = map[status.status] ?? map.unknown;

  return (
    <span className={`${styles.syncBadge} ${info.cls}`} title={info.label}>
      {info.icon}
    </span>
  );
}

interface ProjectRowProps {
  project: Project;
  tracks: Track[];
  onRemove: (slug: string) => void;
}

function ProjectRow({ project, tracks, onRemove }: ProjectRowProps) {
  const counts = trackCountsByStatus(tracks, project.slug);
  return (
    <div className={styles.projectRow} data-tour="project-card">
      <Link to={`/projects/${project.slug}`} className={styles.projectLink}>
        <span className={styles.projectSlug}>{project.slug}</span>
        {project.origin_remote && (
          <span className={styles.projectRemote}>{project.origin_remote}</span>
        )}
        {project.origin_remote && <SyncBadge slug={project.slug} />}
        <span className={styles.trackCounts}>
          {counts.total > 0 ? (
            <>
              <span className={styles.countComplete}>{counts.complete}</span>
              {" / "}
              <span className={styles.countProgress}>{counts["in-progress"]}</span>
              {" / "}
              <span className={styles.countPending}>{counts.pending}</span>
            </>
          ) : (
            <span className={styles.noTracks}>no tracks</span>
          )}
        </span>
      </Link>
      <button
        className={styles.removeBtn}
        onClick={(e) => { e.stopPropagation(); onRemove(project.slug); }}
        title={`Remove ${project.slug}`}
      >
        &times;
      </button>
    </div>
  );
}

export function OverviewPage({ agents, agentsLoading, quota, tracks, onViewLog, onAttach, onSpawnInteractive, spawningInteractive }: OverviewPageProps) {
  const { projects, loading: projectsLoading, adding, removing, error, addProject, removeProject, clearError } = useProjects();
  const { traces } = useTraces();
  const { config, loading: configLoading, updating: configUpdating, updateConfig } = useConfig();
  const tour = useTourContextSafe();
  const [removeSlug, setRemoveSlug] = useState<string | null>(null);
  const [roleFilter, setRoleFilter] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string | null>(null);

  // Wrap addProject to advance tour after successful add
  const handleAddProject = useCallback(async (req: Parameters<typeof addProject>[0]) => {
    const ok = await addProject(req);
    if (ok && tour?.isActive) {
      const step = TOUR_STEPS[tour.currentStep];
      if (step?.id === "add-project") {
        tour.setDemoProjectSlug(req.name ?? req.remote_url.replace(/.*\//, "").replace(/\.git$/, ""));
        tour.nextStep();
      }
    }
    return ok;
  }, [addProject, tour]);

  const filteredAgents = useMemo(() => {
    return agents.filter((a) => {
      if (roleFilter && a.role !== roleFilter) return false;
      if (statusFilter) {
        if (statusFilter === "active") {
          if (a.status !== "running" && a.status !== "waiting") return false;
        } else if (a.status !== statusFilter) {
          return false;
        }
      }
      return true;
    });
  }, [agents, roleFilter, statusFilter]);

  const handleRemoveConfirm = useCallback(
    async (slug: string, cleanup: boolean): Promise<boolean> => {
      const ok = await removeProject(slug, cleanup);
      if (ok) setRemoveSlug(null);
      return ok;
    },
    [removeProject],
  );

  return (
    <>
      <StatCards agentCount={agents.length} quota={quota} />

      <section className={appStyles.panel}>
        <div className={styles.sectionHeader}>
          <h2 className={appStyles.panelTitle}>Agents</h2>
          {onSpawnInteractive && (
            <button
              className={styles.spawnBtn}
              onClick={onSpawnInteractive}
              disabled={spawningInteractive}
            >
              {spawningInteractive ? "Starting..." : "Start Interactive Agent"}
            </button>
          )}
        </div>
        <div className={styles.filterRow}>
          <div className={styles.filterGroup}>
            {["developer", "reviewer", "interactive"].map((role) => (
              <button
                key={role}
                className={`${styles.chip} ${roleFilter === role ? styles.chipActive : ""}`}
                onClick={() => setRoleFilter(roleFilter === role ? null : role)}
              >
                {role}
              </button>
            ))}
          </div>
          <div className={styles.filterGroup}>
            {[
              { key: "active", label: "Active" },
              { key: "completed", label: "Completed" },
              { key: "failed", label: "Failed" },
            ].map(({ key, label }) => (
              <button
                key={key}
                className={`${styles.chip} ${statusFilter === key ? styles.chipActive : ""}`}
                onClick={() => setStatusFilter(statusFilter === key ? null : key)}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        {agentsLoading ? (
          <p className={appStyles.empty}>Loading agents...</p>
        ) : (
          <AgentGrid agents={filteredAgents} onViewLog={onViewLog} onAttach={onAttach} />
        )}
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Projects</h2>
        <AddProjectForm adding={adding} error={error} onAdd={handleAddProject} onClearError={clearError} />
        {projectsLoading ? (
          <p className={appStyles.empty}>Loading projects...</p>
        ) : projects.length === 0 ? (
          <p className={appStyles.empty}>
            No projects registered yet. Use the form above or run <code>kf add &lt;remote&gt;</code>
          </p>
        ) : (
          <div className={styles.projectList}>
            <div className={styles.projectHeader}>
              <span>Project</span>
              <span>Remote</span>
              <span className={styles.syncHeader}>Sync</span>
              <span className={styles.trackCountsHeader}>done / active / pending</span>
              <span className={styles.actionsHeader}></span>
            </div>
            {projects.map((p) => (
              <ProjectRow key={p.slug} project={p} tracks={tracks} onRemove={setRemoveSlug} />
            ))}
          </div>
        )}
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>All Tracks</h2>
        <TrackList tracks={tracks} />
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Traces</h2>
        <TraceList
          traces={traces}
          config={config}
          configLoading={configLoading}
          configUpdating={configUpdating}
          onUpdateConfig={updateConfig}
        />
      </section>

      {removeSlug && (
        <RemoveProjectDialog
          slug={removeSlug}
          removing={removing === removeSlug}
          onConfirm={handleRemoveConfirm}
          onCancel={() => setRemoveSlug(null)}
        />
      )}
    </>
  );
}
