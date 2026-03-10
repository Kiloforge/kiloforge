import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useCallback } from "react";
import type { Track, PaginatedResponse, SSEEventData } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { usePaginatedList, type UsePaginatedListResult } from "./usePaginatedList";

interface UseTracksResult extends UsePaginatedListResult<Track> {
  tracks: Track[];
  loading: boolean;
  handleTrackUpdate: (raw: unknown) => void;
  handleTrackRemoved: (raw: unknown) => void;
}

type InfiniteTracks = InfiniteData<PaginatedResponse<Track>>;

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
      const event = raw as SSEEventData;
      const data = event.data as { id: string; status: string; title?: string; project?: string };
      queryClient.setQueryData<InfiniteTracks>(qk, (prev) => {
        if (!prev) return prev;
        let found = false;
        const pages = prev.pages.map((page) => {
          const idx = page.items.findIndex((t) => t.id === data.id);
          if (idx >= 0) {
            found = true;
            const items = [...page.items];
            items[idx] = { ...items[idx], ...data };
            return { ...page, items };
          }
          return page;
        });
        if (found) return { ...prev, pages };
        queryClient.invalidateQueries({ queryKey: qk });
        return prev;
      });
    },
    [queryClient, qk],
  );

  const handleTrackRemoved = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as { id: string };
      queryClient.setQueryData<InfiniteTracks>(qk, (prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          pages: prev.pages.map((page) => ({
            ...page,
            items: page.items.filter((t) => t.id !== data.id),
            total_count: Math.max(0, page.total_count - (page.items.some((t) => t.id === data.id) ? 1 : 0)),
          })),
        };
      });
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
