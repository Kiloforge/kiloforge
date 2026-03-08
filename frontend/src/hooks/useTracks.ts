import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { Track, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseTracksResult {
  tracks: Track[];
  loading: boolean;
  handleTrackUpdate: (raw: unknown) => void;
  handleTrackRemoved: (raw: unknown) => void;
}

export function useTracks(project?: string): UseTracksResult {
  const queryClient = useQueryClient();
  const key = queryKeys.tracks(project);

  const { data: tracks = [], isLoading } = useQuery({
    queryKey: key,
    queryFn: () => {
      const url = project
        ? `/api/tracks?project=${encodeURIComponent(project)}`
        : "/api/tracks";
      return fetcher<Track[]>(url).then((d) => d || []);
    },
  });

  const handleTrackUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as { id: string; status: string; title?: string; project?: string };
      queryClient.setQueryData<Track[]>(key, (prev = []) => {
        const idx = prev.findIndex((t) => t.id === data.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = { ...next[idx], ...data };
          return next;
        }
        return [...prev, { id: data.id, title: data.title ?? data.id, status: data.status, project: data.project }];
      });
    },
    [queryClient, key],
  );

  const handleTrackRemoved = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as { id: string };
      queryClient.setQueryData<Track[]>(key, (prev = []) =>
        prev.filter((t) => t.id !== data.id),
      );
    },
    [queryClient, key],
  );

  return { tracks, loading: isLoading, handleTrackUpdate, handleTrackRemoved };
}
