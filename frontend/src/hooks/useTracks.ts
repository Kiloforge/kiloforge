import { useState, useEffect, useCallback } from "react";
import type { Track, SSEEventData } from "../types/api";

interface UseTracksResult {
  tracks: Track[];
  loading: boolean;
  handleTrackUpdate: (raw: unknown) => void;
  handleTrackRemoved: (raw: unknown) => void;
}

export function useTracks(project?: string): UseTracksResult {
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const url = project
      ? `/-/api/tracks?project=${encodeURIComponent(project)}`
      : "/-/api/tracks";
    fetch(url)
      .then((r) => r.json())
      .then((data: Track[]) => {
        setTracks(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [project]);

  const handleTrackUpdate = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const data = event.data as { id: string; status: string; title?: string; project?: string };
    setTracks((prev) => {
      const idx = prev.findIndex((t) => t.id === data.id);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = { ...next[idx], ...data };
        return next;
      }
      return [...prev, { id: data.id, title: data.title ?? data.id, status: data.status, project: data.project }];
    });
  }, []);

  const handleTrackRemoved = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const data = event.data as { id: string };
    setTracks((prev) => prev.filter((t) => t.id !== data.id));
  }, []);

  return { tracks, loading, handleTrackUpdate, handleTrackRemoved };
}
