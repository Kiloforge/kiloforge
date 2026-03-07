import { useState, useEffect } from "react";
import type { Track } from "../types/api";

interface UseTracksResult {
  tracks: Track[];
  loading: boolean;
}

export function useTracks(project?: string): UseTracksResult {
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchTracks = () => {
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
    };

    fetchTracks();
    const interval = setInterval(fetchTracks, 30000);
    return () => clearInterval(interval);
  }, [project]);

  return { tracks, loading };
}
