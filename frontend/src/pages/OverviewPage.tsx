import { Link } from "react-router-dom";
import type { Agent, QuotaResponse, StatusResponse, Track } from "../types/api";
import type { Project } from "../types/api";
import { useProjects } from "../hooks/useProjects";
import { StatCards } from "../components/StatCards";
import { AgentGrid } from "../components/AgentGrid";
import { TrackList } from "../components/TrackList";
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

function ProjectRow({ project, tracks }: { project: Project; tracks: Track[] }) {
  const counts = trackCountsByStatus(tracks, project.slug);
  return (
    <Link to={`/projects/${project.slug}`} className={styles.projectRow}>
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
  );
}

export function OverviewPage({ agents, agentsLoading, quota, status, tracks, onViewLog }: OverviewPageProps) {
  const { projects, loading: projectsLoading } = useProjects();

  return (
    <>
      <StatCards agentCount={agents.length} quota={quota} />

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Agents</h2>
        {agentsLoading ? (
          <p className={appStyles.empty}>Loading agents...</p>
        ) : (
          <AgentGrid agents={agents} giteaURL={status?.gitea_url ?? ""} onViewLog={onViewLog} />
        )}
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Projects</h2>
        {projectsLoading ? (
          <p className={appStyles.empty}>Loading projects...</p>
        ) : projects.length === 0 ? (
          <p className={appStyles.empty}>
            No projects registered — run <code>crelay add &lt;remote&gt;</code>
          </p>
        ) : (
          <div className={styles.projectList}>
            <div className={styles.projectHeader}>
              <span>Project</span>
              <span>Remote</span>
              <span className={styles.trackCountsHeader}>done / active / pending</span>
            </div>
            {projects.map((p) => (
              <ProjectRow key={p.slug} project={p} tracks={tracks} />
            ))}
          </div>
        )}
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>All Tracks</h2>
        <TrackList tracks={tracks} />
      </section>
    </>
  );
}
