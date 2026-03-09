import { useQuery } from "@tanstack/react-query";
import type { DiffResponse, BranchInfo } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

export function useProjectDiff(slug: string, branch: string) {
  return useQuery({
    queryKey: queryKeys.projectDiff(slug, branch),
    queryFn: () =>
      fetcher<DiffResponse>(
        `/api/projects/${encodeURIComponent(slug)}/diff?branch=${encodeURIComponent(branch)}`,
      ),
    enabled: !!slug && !!branch,
  });
}

export function useProjectBranches(slug: string) {
  return useQuery({
    queryKey: queryKeys.projectBranches(slug),
    queryFn: () =>
      fetcher<BranchInfo[]>(
        `/api/projects/${encodeURIComponent(slug)}/branches`,
      ),
    enabled: !!slug,
  });
}
