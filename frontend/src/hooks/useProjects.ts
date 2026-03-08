import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import type { Project, AddProjectRequest, SSEEventData, SSHKeyInfo } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { fetcher, FetchError } from "../api/fetcher";

interface UseProjectsResult {
  projects: Project[];
  loading: boolean;
  adding: boolean;
  removing: string | null;
  error: string | null;
  addProject: (req: AddProjectRequest) => Promise<boolean>;
  removeProject: (slug: string, cleanup: boolean) => Promise<boolean>;
  clearError: () => void;
  handleProjectUpdate: (raw: unknown) => void;
  handleProjectRemoved: (raw: unknown) => void;
}

export function useProjects(): UseProjectsResult {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const { data: projects = [], isLoading } = useQuery({
    queryKey: queryKeys.projects,
    queryFn: () => fetcher<Project[]>("/api/projects").then((d) => d || []),
  });

  const addMutation = useMutation({
    mutationFn: (req: AddProjectRequest) =>
      fetcher<Project>("/api/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }),
    onError: (err) => {
      if (err instanceof FetchError) {
        const body = err.body as { error?: string };
        setError(body?.error || `Error ${err.status}`);
      } else {
        setError("Network error");
      }
    },
  });

  const addProject = async (req: AddProjectRequest): Promise<boolean> => {
    setError(null);
    try {
      await addMutation.mutateAsync(req);
      return true;
    } catch {
      return false;
    }
  };

  const removeMutation = useMutation({
    mutationFn: async ({ slug, cleanup }: { slug: string; cleanup: boolean }) => {
      const url = `/api/projects/${encodeURIComponent(slug)}${cleanup ? "?cleanup=true" : ""}`;
      await fetcher<void>(url, { method: "DELETE" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.projects });
    },
    onError: (err) => {
      if (err instanceof FetchError) {
        const body = err.body as { error?: string };
        setError(body?.error || `Error ${err.status}`);
      } else {
        setError("Network error");
      }
    },
  });

  const removeProject = async (slug: string, cleanup: boolean): Promise<boolean> => {
    setError(null);
    try {
      await removeMutation.mutateAsync({ slug, cleanup });
      return true;
    } catch {
      return false;
    }
  };

  const handleProjectUpdate = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as Project;
      if (!data?.slug) return;
      queryClient.setQueryData<Project[]>(queryKeys.projects, (prev = []) => {
        const idx = prev.findIndex((p) => p.slug === data.slug);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = { ...next[idx], ...data };
          return next;
        }
        return [...prev, data];
      });
    },
    [queryClient],
  );

  const handleProjectRemoved = useCallback(
    (raw: unknown) => {
      const event = raw as SSEEventData;
      const data = event.data as { slug: string };
      if (!data?.slug) return;
      queryClient.setQueryData<Project[]>(queryKeys.projects, (prev = []) =>
        prev.filter((p) => p.slug !== data.slug),
      );
    },
    [queryClient],
  );

  const clearError = useCallback(() => setError(null), []);

  return {
    projects,
    loading: isLoading,
    adding: addMutation.isPending,
    removing: removeMutation.isPending ? (removeMutation.variables?.slug ?? null) : null,
    error,
    addProject,
    removeProject,
    clearError,
    handleProjectUpdate,
    handleProjectRemoved,
  };
}

export function useSSHKeys() {
  const { data, isLoading, refetch } = useQuery({
    queryKey: queryKeys.sshKeys,
    queryFn: () => fetcher<{ keys: SSHKeyInfo[] }>("/api/ssh-keys").then((d) => d.keys || []),
    enabled: false,
  });

  return {
    keys: data ?? [],
    loading: isLoading,
    fetchKeys: () => { refetch(); },
  };
}
