import type { Track } from "../types/api";

export function isTrackReady(track: Track): boolean {
  if (track.status !== "pending") return false;
  return !track.deps_count || track.deps_count === track.deps_met;
}
