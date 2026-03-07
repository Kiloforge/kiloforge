import { useState, useEffect } from "react";
import type { Project } from "../types/api";

interface UseProjectsResult {
  projects: Project[];
  loading: boolean;
}

export function useProjects(): UseProjectsResult {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/-/api/projects")
      .then((r) => r.json())
      .then((data: Project[]) => {
        setProjects(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  return { projects, loading };
}
