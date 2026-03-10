import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { ProjectSettings, UpdateProjectSettingsRequest } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher } from "../api/fetcher";

interface UseProjectSettingsResult {
  settings: ProjectSettings | null;
  loading: boolean;
  updating: boolean;
  updateSettings: (req: UpdateProjectSettingsRequest) => Promise<boolean>;
}

export function useProjectSettings(slug: string | undefined): UseProjectSettingsResult {
  const queryClient = useQueryClient();

  const { data: settings = null, isLoading } = useQuery({
    queryKey: queryKeys.projectSettings(slug ?? ""),
    queryFn: () =>
      fetcher<ProjectSettings>(
        `/api/projects/${encodeURIComponent(slug!)}/settings`,
      ),
    enabled: !!slug,
  });

  const mutation = useMutation({
    mutationFn: (req: UpdateProjectSettingsRequest) =>
      fetcher<ProjectSettings>(
        `/api/projects/${encodeURIComponent(slug!)}/settings`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        },
      ),
    onSuccess: (data) => {
      queryClient.setQueryData<ProjectSettings>(
        queryKeys.projectSettings(slug!),
        data,
      );
    },
  });

  const updateSettings = async (req: UpdateProjectSettingsRequest): Promise<boolean> => {
    try {
      await mutation.mutateAsync(req);
      return true;
    } catch {
      return false;
    }
  };

  return { settings, loading: isLoading, updating: mutation.isPending, updateSettings };
}
