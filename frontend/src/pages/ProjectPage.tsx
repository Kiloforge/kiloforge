import { Link, useParams } from "react-router-dom";
import { useTracks } from "../hooks/useTracks";
import { useProjects } from "../hooks/useProjects";
import { useBoard } from "../hooks/useBoard";
import { TrackList } from "../components/TrackList";
import { KanbanBoard } from "../components/KanbanBoard";
import appStyles from "../App.module.css";
import styles from "./ProjectPage.module.css";

export function ProjectPage() {
  const { slug } = useParams<{ slug: string }>();
  const { tracks, loading: tracksLoading } = useTracks(slug);
  const { projects } = useProjects();
  const { board, loading: boardLoading, moveCard } = useBoard(slug);
  const project = projects.find((p) => p.slug === slug);

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

      <section className={appStyles.panel}>
        <h2 className={appStyles.panelTitle}>Board</h2>
        {boardLoading ? (
          <p className={appStyles.empty}>Loading board...</p>
        ) : board && Object.keys(board.cards).length > 0 ? (
          <KanbanBoard board={board} onMoveCard={moveCard} />
        ) : (
          <p className={appStyles.empty}>No cards on the board yet. Run sync to populate.</p>
        )}
      </section>

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
