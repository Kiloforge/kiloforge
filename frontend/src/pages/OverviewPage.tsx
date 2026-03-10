import { useState, useMemo, useCallback } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import type { Agent, QuotaResponse, Track, Project, SyncStatus, SwarmStatus, SwarmSettings } from "../types/api";
import { useProjects } from "../hooks/useProjects";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { MetricsPanel } from "../components/MetricsPanel";
import { AgentGrid } from "../components/AgentGrid";
import { TrackList } from "../components/TrackList";
import { TraceList } from "../components/TraceList";
import { AddProjectForm } from "../components/AddProjectForm";
import { RemoveProjectDialog } from "../components/RemoveProjectDialog";
import { SwarmPanel } from "../components/SwarmPanel";
import { PaginatedList } from "../components/PaginatedList";
import { InlineSpinner } from "../components/InlineSpinner";
import { GettingStartedChecklist } from "../components/GettingStartedChecklist";
import { HelpTooltip } from "../components/HelpTooltip";
import { AdvisorHub } from "../components/AdvisorHub";
import type { AgentRole } from "../components/AgentLauncher";
import { useTraces } from "../hooks/useTraces";
import styles from "./OverviewPage.module.css";
import appStyles from "../App.module.css";

interface OverviewPageProps {
  agents: Agent[];
  agentsLoading: boolean;
  agentRemainingCount?: number;
  agentHasNextPage?: boolean;
  agentFetchingNextPage?: boolean;
  onAgentLoadMore?: () => void;
  quota: QuotaResponse | null;
  tracks: Track[];
  onViewLog: (agentId: string) => void;
  onAttach?: (agentId: string) => void;
  onSpawnInteractive?: () => void;
  spawningInteractive?: boolean;
  swarm?: SwarmStatus | null;
  swarmLoading?: boolean;
  swarmStarting?: boolean;
  swarmStopping?: boolean;
  swarmUpdatingSettings?: boolean;
  onSwarmStart?: () => void;
  onSwarmStop?: () => void;
  onSwarmUpdateSettings?: (settings: SwarmSettings) => void;
  trackRemainingCount?: number;
  trackHasNextPage?: boolean;
  trackFetchingNextPage?: boolean;
  onTrackLoadMore?: () => void;
  onAdvisorLaunch?: (role: AgentRole, prompt: string, project?: string) => void;
  advisorLaunching?: boolean;
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

export function OverviewPage({ agents, agentsLoading, agentRemainingCount = 0, agentHasNextPage = false, agentFetchingNextPage = false, onAgentLoadMore, quota, tracks, onViewLog, onAttach, onSpawnInteractive, spawningInteractive, swarm, swarmLoading, swarmStarting, swarmStopping, swarmUpdatingSettings, onSwarmStart, onSwarmStop, onSwarmUpdateSettings, trackRemainingCount = 0, trackHasNextPage = false, trackFetchingNextPage = false, onTrackLoadMore, onAdvisorLaunch, advisorLaunching }: OverviewPageProps) {
  const { projects, loading: projectsLoading, adding, removing, error, addProject, removeProject, clearError } = useProjects();
  const { traces, remainingCount: traceRemainingCount, hasNextPage: traceHasNextPage, isFetchingNextPage: traceFetchingNextPage, fetchNextPage: traceFetchNextPage } = useTraces();
  const [removeSlug, setRemoveSlug] = useState<string | null>(null);
  const [roleFilter, setRoleFilter] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string | null>(null);

  const filteredAgents = useMemo(() => {
    return agents.filter((a) => {
      if (roleFilter) {
        if (roleFilter === "advisor") {
          if (!a.role.startsWith("advisor-")) return false;
        } else if (a.role !== roleFilter) {
          return false;
        }
      }
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
      <MetricsPanel agentCount={agents.length} quota={quota} />

      <GettingStartedChecklist projects={projects} agents={agents} tracks={tracks} />

      <section className={appStyles.panel}>
        <div className={styles.sectionHeader}>
          <h2 className={appStyles.panelTitle}>
            Agents
            <HelpTooltip term="Agents" definition="AI workers that implement tracks. Each agent runs in its own worktree and can be a developer, reviewer, or interactive session." />
          </h2>
          <div className={styles.sectionActions}>
            <Link to="/agents" className={styles.viewAllLink}>View all</Link>
            {onSpawnInteractive && (
              <button
                className={styles.spawnBtn}
                onClick={onSpawnInteractive}
                disabled={spawningInteractive}
              >
                {spawningInteractive ? "Starting..." : "New Agent"}
              </button>
            )}
          </div>
        </div>
        <div className={styles.filterRow}>
          <div className={styles.filterGroup}>
            {["developer", "reviewer", "interactive", "advisor"].map((role) => (
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
          <InlineSpinner label="Loading agents..." />
        ) : (
          <PaginatedList
            remainingCount={agentRemainingCount}
            hasNextPage={agentHasNextPage}
            isFetchingNextPage={agentFetchingNextPage}
            onLoadMore={onAgentLoadMore ?? (() => {})}
          >
            <AgentGrid agents={filteredAgents} onViewLog={onViewLog} onAttach={onAttach} />
          </PaginatedList>
        )}
      </section>

      {onAdvisorLaunch && (
        <AdvisorHub
          agents={agents}
          onLaunch={onAdvisorLaunch}
          launching={advisorLaunching}
          onViewLog={onViewLog}
          onAttach={onAttach}
        />
      )}

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>
          Projects
          <HelpTooltip term="Projects" definition="Registered Git repositories that Kiloforge manages. Each project can have tracks generated and agents assigned." />
        </h2>
        <AddProjectForm adding={adding} error={error} onAdd={addProject} onClearError={clearError} />
        {projectsLoading ? (
          <InlineSpinner label="Loading projects..." />
        ) : projects.length === 0 ? (
          <div className={appStyles.empty}>
            <p>No projects registered yet, Kiloforger.</p>
            <p style={{ marginTop: 8, fontSize: 12, color: "var(--text-dimmed)" }}>
              Use the form above, or from the terminal:
            </p>
            <code style={{ display: "block", marginTop: 4, padding: "6px 10px", background: "var(--bg-code)", borderRadius: 6, fontSize: 12 }}>
              kf add https://github.com/you/repo.git
            </code>
          </div>
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
        <h2 className={appStyles.panelTitle}>
          AI Agent Swarm
          <HelpTooltip term="Swarm" definition="A managed pool of AI agents that automatically pick up and implement tracks. Start the swarm to parallelize development work." />
        </h2>
        <SwarmPanel
          swarm={swarm ?? null}
          loading={swarmLoading ?? false}
          starting={swarmStarting ?? false}
          stopping={swarmStopping ?? false}
          updatingSettings={swarmUpdatingSettings ?? false}
          onStart={onSwarmStart ?? (() => {})}
          onStop={onSwarmStop ?? (() => {})}
          onUpdateSettings={onSwarmUpdateSettings ?? (() => {})}
        />
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>
          All Tracks
          <HelpTooltip term="Tracks" definition="Units of work generated from feature requests. Each track has a spec, implementation plan, and phases that agents execute sequentially." />
        </h2>
        <PaginatedList
          remainingCount={trackRemainingCount}
          hasNextPage={trackHasNextPage}
          isFetchingNextPage={trackFetchingNextPage}
          onLoadMore={onTrackLoadMore ?? (() => {})}
        >
          <TrackList tracks={tracks} />
        </PaginatedList>
      </section>

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Traces</h2>
        <PaginatedList
          remainingCount={traceRemainingCount}
          hasNextPage={traceHasNextPage}
          isFetchingNextPage={traceFetchingNextPage}
          onLoadMore={() => traceFetchNextPage()}
        >
          <TraceList traces={traces} />
        </PaginatedList>
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
