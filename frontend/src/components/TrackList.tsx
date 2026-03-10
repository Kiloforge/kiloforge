import { Link } from "react-router-dom";
import type { Track } from "../types/api";
import styles from "./TrackList.module.css";

function statusIcon(status: string): string {
  switch (status) {
    case "complete":
      return "\u2713";
    case "in-progress":
      return "\u25B6";
    default:
      return "\u25CB";
  }
}

interface TrackListProps {
  tracks: Track[];
  projectSlug?: string;
}

export function TrackList({ tracks, projectSlug }: TrackListProps) {
  if (tracks.length === 0) {
    return (
      <div className={styles.empty}>
        <p>No tracks found</p>
        <p className={styles.hint}>Tracks are generated when an architect agent analyzes a feature request.</p>
      </div>
    );
  }

  return (
    <div className={styles.list}>
      {tracks.map((track) => {
        const slug = projectSlug || track.project;
        const content = (
          <>
            <span className={`${styles.status} ${styles[track.status] ?? ""}`}>
              {statusIcon(track.status)}
            </span>
            <span className={styles.id}>{track.id}</span>
            <span className={styles.title}>{track.title}</span>
          </>
        );

        if (slug) {
          return (
            <Link
              key={track.id}
              to={`/projects/${slug}/tracks/${track.id}`}
              className={styles.row}
            >
              {content}
            </Link>
          );
        }

        return (
          <div key={track.id} className={styles.row}>
            {content}
          </div>
        );
      })}
    </div>
  );
}
