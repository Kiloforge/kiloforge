import { useState, useEffect, useCallback } from "react";
import type { Project, AddProjectRequest, SSEEventData } from "../types/api";

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
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [removing, setRemoving] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchProjects = useCallback(() => {
    fetch("/-/api/projects")
      .then((r) => r.json())
      .then((data: Project[]) => {
        setProjects(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const handleProjectUpdate = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const data = event.data as Project;
    if (!data?.slug) return;
    setProjects((prev) => {
      const idx = prev.findIndex((p) => p.slug === data.slug);
      if (idx >= 0) {
        const next = [...prev];
        next[idx] = { ...next[idx], ...data };
        return next;
      }
      return [...prev, data];
    });
  }, []);

  const handleProjectRemoved = useCallback((raw: unknown) => {
    const event = raw as SSEEventData;
    const data = event.data as { slug: string };
    if (!data?.slug) return;
    setProjects((prev) => prev.filter((p) => p.slug !== data.slug));
  }, []);

  const addProject = useCallback(
    async (req: AddProjectRequest): Promise<boolean> => {
      setAdding(true);
      setError(null);
      try {
        const resp = await fetch("/-/api/projects", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        });
        if (!resp.ok) {
          const body = await resp.json().catch(() => ({ error: "Request failed" }));
          setError(body.error || `Error ${resp.status}`);
          setAdding(false);
          return false;
        }
        // SSE will update the list; no need to refetch.
        setAdding(false);
        return true;
      } catch {
        setError("Network error");
        setAdding(false);
        return false;
      }
    },
    [],
  );

  const removeProject = useCallback(
    async (slug: string, cleanup: boolean): Promise<boolean> => {
      setRemoving(slug);
      setError(null);
      try {
        const url = `/-/api/projects/${encodeURIComponent(slug)}${cleanup ? "?cleanup=true" : ""}`;
        const resp = await fetch(url, { method: "DELETE" });
        if (!resp.ok) {
          const body = await resp.json().catch(() => ({ error: "Request failed" }));
          setError(body.error || `Error ${resp.status}`);
          setRemoving(null);
          return false;
        }
        // SSE will update the list; no need to refetch.
        setRemoving(null);
        return true;
      } catch {
        setError("Network error");
        setRemoving(null);
        return false;
      }
    },
    [],
  );

  const clearError = useCallback(() => setError(null), []);

  return { projects, loading, adding, removing, error, addProject, removeProject, clearError, handleProjectUpdate, handleProjectRemoved };
}
