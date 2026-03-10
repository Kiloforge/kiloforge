import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import type { Track, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { usePaginatedList, type UsePaginatedListResult } from "./usePaginatedList";

interface UseTracksResult extends UsePaginatedListResult<Track> {
  tracks: Track[];
  loading: boolean;
  handleTrackUpdate: (raw: unknown) => void;
  handleTrackRemoved: (raw: unknown) => void;
}

export function useTracks(project?: string): UseTracksResult {
  const queryClient = useQueryClient();
  const qk = queryKeys.tracksPaginated(project);

  const paginated = usePaginatedList<Track>({
    queryKey: qk,
    url: "/api/tracks",
    params: project ? { project } : undefined,
  });

  const handleTrackUpdate = useCallback(
    (raw: unknown) => {
      const _event = raw as SSEEventData;
      queryClient.invalidateQueries({ queryKey: qk });
    },
    [queryClient, qk],
  );

  const handleTrackRemoved = useCallback(
    (raw: unknown) => {
      const _event = raw as SSEEventData;
      queryClient.invalidateQueries({ queryKey: qk });
    },
    [queryClient, qk],
  );

  return {
    ...paginated,
    tracks: paginated.items,
    loading: paginated.isLoading,
    handleTrackUpdate,
    handleTrackRemoved,
  };
}
