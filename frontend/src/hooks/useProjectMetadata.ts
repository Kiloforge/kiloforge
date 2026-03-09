import { useQuery } from "@tanstack/react-query";
import type { ProjectMetadata } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

export function useProjectMetadata(slug: string | undefined) {
  return useQuery({
    queryKey: queryKeys.projectMetadata(slug ?? ""),
    queryFn: () =>
      fetcher<ProjectMetadata>(
        `/api/projects/${encodeURIComponent(slug!)}/metadata`,
      ),
    enabled: !!slug,
  });
}
