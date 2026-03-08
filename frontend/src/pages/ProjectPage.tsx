import { useCallback, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useTracks } from "../hooks/useTracks";
import { useProjects } from "../hooks/useProjects";
import { useBoard } from "../hooks/useBoard";
import { useOriginSync } from "../hooks/useOriginSync";
import { TrackList } from "../components/TrackList";
import { KanbanBoard } from "../components/KanbanBoard";
import { SyncPanel } from "../components/SyncPanel";
import { AgentTerminal } from "../components/AgentTerminal";
import { AdminPanel } from "../components/AdminPanel";
import { ConsentDialog } from "../components/ConsentDialog";
import { useConsent } from "../hooks/useConsent";
import appStyles from "../App.module.css";
import styles from "./ProjectPage.module.css";

export function ProjectPage() {
  const { slug } = useParams<{ slug: string }>();
  const { tracks, loading: tracksLoading } = useTracks(slug);
  const { projects } = useProjects();
  const { board, loading: boardLoading, moveCard, refresh: refreshBoard } = useBoard(slug);
  const { syncStatus, loading: syncLoading, pushing, pulling, error: syncError, push, pull, refresh: refreshSync, clearError: clearSyncError } = useOriginSync(slug);
  const project = projects.find((p) => p.slug === slug);

  const [showPrompt, setShowPrompt] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [generating, setGenerating] = useState(false);
  const [terminalAgentId, setTerminalAgentId] = useState<string | null>(null);
  const [adminAgentId, setAdminAgentId] = useState<string | null>(null);
  const consent = useConsent();

  const handlePush = useCallback((remoteBranch: string) => {
    push({ remote_branch: remoteBranch });
  }, [push]);

  const handlePull = useCallback((remoteBranch?: string) => {
    pull(remoteBranch);
  }, [pull]);

  const handleGenerateTracks = useCallback(async () => {
    if (!prompt.trim() || !slug) return;
    setGenerating(true);
    try {
      const resp = await fetch("/api/tracks/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt: prompt.trim(), project: slug }),
      });
      if (resp.status === 403) {
        consent.requestConsent(() => handleGenerateTracks());
        return;
      }
      if (resp.ok) {
        const data = await resp.json() as { agent_id: string; ws_url: string };
        setTerminalAgentId(data.agent_id);
        setShowPrompt(false);
        setPrompt("");
      }
    } finally {
      setGenerating(false);
    }
  }, [prompt, slug, consent]);

  const handleDeleteTrack = useCallback(async (trackId: string) => {
    if (!slug) return;
    await fetch(`/api/tracks/${encodeURIComponent(trackId)}?project=${encodeURIComponent(slug)}`, {
      method: "DELETE",
    });
    refreshBoard();
  }, [slug, refreshBoard]);

  const handleTerminalClose = useCallback(() => {
    setTerminalAgentId(null);
    refreshBoard();
  }, [refreshBoard]);

  const handleAdminTerminalClose = useCallback(() => {
    setAdminAgentId(null);
    refreshBoard();
  }, [refreshBoard]);

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
        <div className={styles.boardHeader}>
          <h2 className={appStyles.panelTitle}>Board</h2>
          <button
            className={styles.generateBtn}
            onClick={() => setShowPrompt((v) => !v)}
          >
            Generate Tracks
          </button>
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
                disabled={!prompt.trim() || generating}
                onClick={handleGenerateTracks}
              >
                {generating ? "Starting..." : "Generate"}
              </button>
              <button className={styles.promptCancel} onClick={() => { setShowPrompt(false); setPrompt(""); }}>
                Cancel
              </button>
            </div>
          </div>
        )}
        {boardLoading ? (
          <p className={appStyles.empty}>Loading board...</p>
        ) : board && Object.keys(board.cards).length > 0 ? (
          <KanbanBoard board={board} onMoveCard={moveCard} onDeleteTrack={handleDeleteTrack} />
        ) : (
          <p className={appStyles.empty}>No cards on the board yet. Run sync to populate.</p>
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
          onStartOperation={setAdminAgentId}
        />
      </section>

      {adminAgentId && (
        <AgentTerminal agentId={adminAgentId} onClose={handleAdminTerminalClose} />
      )}

      {consent.showDialog && <ConsentDialog onAccept={consent.accept} onDeny={consent.deny} />}

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Tracks</h2>
        {tracksLoading ? (
          <p className={appStyles.empty}>Loading tracks...</p>
        ) : (
          <TrackList tracks={tracks} />
        )}
      </section>
    </>
  );
}
