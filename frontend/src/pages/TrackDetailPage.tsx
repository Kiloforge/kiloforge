import { useState } from "react";
import { Link, useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { TrackDetail } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";
import { StatusBadge } from "../components/StatusBadge";
import { AgentRegister } from "../components/AgentRegister";
import { TraceList } from "../components/TraceList";
import styles from "./TrackDetailPage.module.css";

export function TrackDetailPage() {
  const { slug, trackId } = useParams<{ slug: string; trackId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [confirmReject, setConfirmReject] = useState(false);

  const { data: track, error, isLoading } = useQuery({
    queryKey: queryKeys.trackDetail(trackId ?? "", slug ?? ""),
    queryFn: () =>
      fetcher<TrackDetail>(
        `/api/tracks/${encodeURIComponent(trackId!)}?project=${encodeURIComponent(slug!)}`,
      ),
    enabled: !!trackId && !!slug,
  });

  const approveMutation = useMutation({
    mutationFn: () =>
      fetcher<void>(`/api/board/${encodeURIComponent(slug!)}/move`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ track_id: trackId, to_column: "approved" }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.board(slug ?? "") });
      navigate(`/projects/${slug}`);
    },
  });

  const rejectMutation = useMutation({
    mutationFn: () =>
      fetcher<void>(
        `/api/tracks/${encodeURIComponent(trackId!)}?project=${encodeURIComponent(slug!)}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.board(slug ?? "") });
      navigate(`/projects/${slug}`);
    },
  });

  const isBacklog = track?.status === "pending";

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

      {isBacklog && (
        <div className={styles.actions}>
          {confirmReject ? (
            <div className={styles.confirmRow}>
              <span className={styles.confirmText}>Delete this track?</span>
              <button
                className={styles.confirmYes}
                onClick={() => rejectMutation.mutate()}
                disabled={rejectMutation.isPending}
              >
                Yes, delete
              </button>
              <button className={styles.confirmNo} onClick={() => setConfirmReject(false)}>
                Cancel
              </button>
            </div>
          ) : (
            <>
              <button
                className={styles.approveBtn}
                onClick={() => approveMutation.mutate()}
                disabled={approveMutation.isPending}
              >
                {approveMutation.isPending ? "Approving..." : "Approve"}
              </button>
              <button
                className={styles.rejectBtn}
                onClick={() => setConfirmReject(true)}
                disabled={rejectMutation.isPending}
              >
                Reject
              </button>
            </>
          )}
        </div>
      )}

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

      {track.agent_register &&
        (track.agent_register.created_by || track.agent_register.claimed_by) && (
        <section className={styles.section}>
          <h3 className={styles.sectionTitle}>Agent Register</h3>
          <AgentRegister register={track.agent_register} />
        </section>
      )}

      {track.traces && track.traces.length > 0 && (
        <section className={styles.section}>
          <h3 className={styles.sectionTitle}>
            Traces
            <span className={styles.sectionCount}>{track.traces.length}</span>
          </h3>
          <div className={styles.tracesWrapper}>
            <TraceList traces={track.traces} />
          </div>
        </section>
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
