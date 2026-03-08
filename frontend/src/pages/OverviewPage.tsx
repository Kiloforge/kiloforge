import { useState, useCallback } from "react";
import { Link } from "react-router-dom";
import type { Agent, QuotaResponse, StatusResponse, Track } from "../types/api";
import type { Project } from "../types/api";
import { useProjects } from "../hooks/useProjects";
import { StatCards } from "../components/StatCards";
import { AgentGrid } from "../components/AgentGrid";
import { TrackList } from "../components/TrackList";
import { TraceList } from "../components/TraceList";
import { AddProjectForm } from "../components/AddProjectForm";
import { RemoveProjectDialog } from "../components/RemoveProjectDialog";
import { useTraces } from "../hooks/useTraces";
import { useConfig } from "../hooks/useConfig";
import styles from "./OverviewPage.module.css";
import appStyles from "../App.module.css";

interface OverviewPageProps {
  agents: Agent[];
  agentsLoading: boolean;
  quota: QuotaResponse | null;
  status: StatusResponse | null;
  tracks: Track[];
  onViewLog: (agentId: string) => void;
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

interface ProjectRowProps {
  project: Project;
  tracks: Track[];
  onRemove: (slug: string) => void;
}

function ProjectRow({ project, tracks, onRemove }: ProjectRowProps) {
  const counts = trackCountsByStatus(tracks, project.slug);
  return (
    <div className={styles.projectRow}>
      <Link to={`/projects/${project.slug}`} className={styles.projectLink}>
        <span className={styles.projectSlug}>{project.slug}</span>
        {project.origin_remote && (
          <span className={styles.projectRemote}>{project.origin_remote}</span>
        )}
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

export function OverviewPage({ agents, agentsLoading, quota, status, tracks, onViewLog }: OverviewPageProps) {
  const { projects, loading: projectsLoading, adding, removing, error, addProject, removeProject, clearError } = useProjects();
  const { traces } = useTraces();
  const { config, loading: configLoading, updating: configUpdating, updateConfig } = useConfig();
  const [removeSlug, setRemoveSlug] = useState<string | null>(null);

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
        <h2 className={appStyles.panelTitle}>Agents</h2>
        {agentsLoading ? (
          <p className={appStyles.empty}>Loading agents...</p>
        ) : (
          <AgentGrid agents={agents} onViewLog={onViewLog} />
        )}
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Projects</h2>
        <AddProjectForm adding={adding} error={error} onAdd={addProject} onClearError={clearError} />
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
