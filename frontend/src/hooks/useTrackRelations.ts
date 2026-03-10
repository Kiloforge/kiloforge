import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetcher } from "../api/fetcher";
import { queryKeys } from "../api/queryKeys";
import type { TrackDetail, TrackDependency, TrackConflict } from "../types/api";

interface UseTrackRelationsResult {
  dependencies: Map<string, TrackDependency[]>;
  conflicts: Map<string, TrackConflict[]>;
  loading: boolean;
}

/**
 * Fetches track detail for all tracks on the board to collect
 * dependency and conflict data for relationship visualization.
 */
export function useTrackRelations(
  trackIds: string[],
  project?: string,
): UseTrackRelationsResult {
  const { data: details = [], isLoading } = useQuery({
    queryKey: [...queryKeys.tracks(project), "relations"] as const,
    queryFn: async () => {
      if (!project || trackIds.length === 0) return [];
      // Fetch detail for each track in parallel
      const results = await Promise.allSettled(
        trackIds.map((id) =>
          fetcher<TrackDetail>(
            `/api/tracks/${encodeURIComponent(id)}?project=${encodeURIComponent(project)}`,
          ),
        ),
      );
      return results
        .filter((r): r is PromiseFulfilledResult<TrackDetail> => r.status === "fulfilled")
        .map((r) => r.value);
    },
    enabled: !!project && trackIds.length > 0,
    staleTime: 60_000,
  });

  const dependencies = useMemo(() => {
    const map = new Map<string, TrackDependency[]>();
    for (const detail of details) {
      if (detail?.dependencies && detail.dependencies.length > 0) {
        map.set(detail.id, detail.dependencies);
      }
    }
    return map;
  }, [details]);

  const conflicts = useMemo(() => {
    const map = new Map<string, TrackConflict[]>();
    for (const detail of details) {
      if (detail?.conflicts && detail.conflicts.length > 0) {
        map.set(detail.id, detail.conflicts);
      }
    }
    return map;
  }, [details]);

  return { dependencies, conflicts, loading: isLoading };
}
