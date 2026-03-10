import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { ConfigResponse, UpdateConfigRequest } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseConfigResult {
  config: ConfigResponse | null;
  loading: boolean;
  updating: boolean;
  updateConfig: (req: UpdateConfigRequest) => Promise<boolean>;
}

export function useConfig(): UseConfigResult {
  const queryClient = useQueryClient();

  const { data: config = null, isLoading } = useQuery({
    queryKey: queryKeys.config,
    queryFn: () => fetcher<ConfigResponse>("/api/config"),
  });

  const mutation = useMutation({
    mutationFn: (req: UpdateConfigRequest) =>
      fetcher<ConfigResponse>("/api/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }),
    onMutate: async (req) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.config });
      const previous = queryClient.getQueryData<ConfigResponse>(queryKeys.config);
      if (previous) {
        queryClient.setQueryData<ConfigResponse>(queryKeys.config, { ...previous, ...req });
      }
      return { previous };
    },
    onError: (_err, _req, context) => {
      if (context?.previous) {
        queryClient.setQueryData<ConfigResponse>(queryKeys.config, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.config });
    },
  });

  const updateConfig = async (req: UpdateConfigRequest): Promise<boolean> => {
    try {
      await mutation.mutateAsync(req);
      return true;
    } catch {
      return false;
    }
  };

  return { config, loading: isLoading, updating: mutation.isPending, updateConfig };
}
