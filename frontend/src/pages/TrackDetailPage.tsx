import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import type { TrackDetail } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { StatusBadge } from "../components/StatusBadge";
import styles from "./TrackDetailPage.module.css";

export function TrackDetailPage() {
  const { slug, trackId } = useParams<{ slug: string; trackId: string }>();

  const { data: track, error, isLoading } = useQuery({
    queryKey: queryKeys.trackDetail(trackId ?? "", slug ?? ""),
    queryFn: () =>
      fetcher<TrackDetail>(
        `/api/tracks/${encodeURIComponent(trackId!)}?project=${encodeURIComponent(slug!)}`,
      ),
    enabled: !!trackId && !!slug,
  });

  if (isLoading) {
    return (
      <div className={styles.page}>
        <Link to={`/projects/${slug}`} className={styles.back}>&larr; Back to project</Link>
        <p className={styles.loading}>Loading track...</p>
      </div>
    );
  }

  if (error || !track) {
    return (
      <div className={styles.page}>
        <Link to={`/projects/${slug}`} className={styles.back}>&larr; Back to project</Link>
        <p className={styles.error}>{error?.message ?? "Track not found"}</p>
      </div>
    );
  }

  const phasesTotal = track.phases_total ?? 0;
  const phasesCompleted = track.phases_completed ?? 0;
  const tasksTotal = track.tasks_total ?? 0;
  const tasksCompleted = track.tasks_completed ?? 0;

  return (
    <div className={styles.page}>
      <div className={styles.topBar}>
        <Link to={`/projects/${slug}`} className={styles.back}>&larr; Back to project</Link>
      </div>

      <div className={styles.header}>
        <h2 className={styles.title}>{track.title}</h2>
        <div className={styles.badges}>
          <StatusBadge status={track.status} />
          {track.type && <span className={styles.typeBadge}>{track.type}</span>}
        </div>
      </div>

      <div className={styles.metaGrid}>
        <div className={styles.metaItem}>
          <span className={styles.metaLabel}>Track ID</span>
          <span className={styles.mono}>{track.id}</span>
        </div>
        {phasesTotal > 0 && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Phases</span>
            <span>{phasesCompleted} / {phasesTotal}</span>
          </div>
        )}
        {tasksTotal > 0 && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Tasks</span>
            <span>{tasksCompleted} / {tasksTotal}</span>
          </div>
        )}
        {track.created_at && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Created</span>
            <span>{new Date(track.created_at).toLocaleString()}</span>
          </div>
        )}
        {track.updated_at && (
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Updated</span>
            <span>{new Date(track.updated_at).toLocaleString()}</span>
          </div>
        )}
      </div>

      {(phasesTotal > 0 || tasksTotal > 0) && (
        <div className={styles.progressSection}>
          {tasksTotal > 0 && (
            <div className={styles.progressBar}>
              <div
                className={styles.progressFill}
                style={{ width: `${(tasksCompleted / tasksTotal) * 100}%` }}
              />
            </div>
          )}
          <span className={styles.progressLabel}>
            {tasksCompleted}/{tasksTotal} tasks complete
          </span>
        </div>
      )}

      {track.spec && (
        <section className={styles.section}>
          <h3 className={styles.sectionTitle}>Specification</h3>
          <pre className={styles.markdown}>{track.spec}</pre>
        </section>
      )}

      {track.plan && (
        <section className={styles.section}>
          <h3 className={styles.sectionTitle}>Implementation Plan</h3>
          <pre className={styles.markdown}>{track.plan}</pre>
        </section>
      )}
    </div>
  );
}
