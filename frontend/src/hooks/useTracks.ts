import { useState, useEffect } from "react";
import type { Track } from "../types/api";

interface UseTracksResult {
  tracks: Track[];
  loading: boolean;
}

export function useTracks(): UseTracksResult {
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchTracks = () => {
      fetch("/-/api/tracks")
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
  }, []);

  return { tracks, loading };
}
