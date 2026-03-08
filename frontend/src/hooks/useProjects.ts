import { useState, useEffect, useCallback } from "react";
import type { Project, AddProjectRequest } from "../types/api";

interface UseProjectsResult {
  projects: Project[];
  loading: boolean;
  adding: boolean;
  removing: string | null;
  error: string | null;
  addProject: (req: AddProjectRequest) => Promise<boolean>;
  removeProject: (slug: string, cleanup: boolean) => Promise<boolean>;
  clearError: () => void;
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
        fetchProjects();
        setAdding(false);
        return true;
      } catch {
        setError("Network error");
        setAdding(false);
        return false;
      }
    },
    [fetchProjects],
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
        fetchProjects();
        setRemoving(null);
        return true;
      } catch {
        setError("Network error");
        setRemoving(null);
        return false;
      }
    },
    [fetchProjects],
  );

  const clearError = useCallback(() => setError(null), []);

  return { projects, loading, adding, removing, error, addProject, removeProject, clearError };
}
