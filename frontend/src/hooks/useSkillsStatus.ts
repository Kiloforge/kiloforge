import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { SkillsStatus } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseSkillsStatusResult {
  status: SkillsStatus | null;
  loading: boolean;
  updating: boolean;
  triggerUpdate: (force?: boolean) => Promise<void>;
  refresh: () => void;
}

export function useSkillsStatus(): UseSkillsStatusResult {
  const queryClient = useQueryClient();

  const { data: status = null, isLoading, refetch } = useQuery({
    queryKey: queryKeys.skills,
    queryFn: () => fetcher<SkillsStatus>("/api/skills"),
    refetchInterval: 60_000,
  });

  const mutation = useMutation({
    mutationFn: (force: boolean) =>
      fetcher<void>("/api/skills/update", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ force }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.skills });
    },
  });

  const triggerUpdate = async (force = false) => {
    await mutation.mutateAsync(force);
  };

  return {
    status,
    loading: isLoading,
    updating: mutation.isPending,
    triggerUpdate,
    refresh: () => { refetch(); },
  };
}
