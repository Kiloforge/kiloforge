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
    onSuccess: (data) => {
      queryClient.setQueryData<ConfigResponse>(queryKeys.config, data);
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
