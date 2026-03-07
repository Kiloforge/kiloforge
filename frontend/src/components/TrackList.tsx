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

export function TrackList({ tracks }: { tracks: Track[] }) {
  if (tracks.length === 0) {
    return <p className={styles.empty}>No tracks found</p>;
  }

  return (
    <div className={styles.list}>
      {tracks.map((track) => (
        <div key={track.id} className={styles.row}>
          <span className={`${styles.status} ${styles[track.status] ?? ""}`}>
            {statusIcon(track.status)}
          </span>
          <span className={styles.id}>{track.id}</span>
          <span className={styles.title}>{track.title}</span>
        </div>
      ))}
    </div>
  );
}
