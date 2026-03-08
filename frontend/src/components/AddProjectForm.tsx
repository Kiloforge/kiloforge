import { useState, useCallback } from "react";
import type { AddProjectRequest } from "../types/api";
import styles from "./AddProjectForm.module.css";

interface AddProjectFormProps {
  adding: boolean;
  error: string | null;
  onAdd: (req: AddProjectRequest) => Promise<boolean>;
  onClearError: () => void;
}

const URL_PATTERN = /^(https?:\/\/.+|ssh:\/\/.+|[^/]+@[^:]+:.+)$/;

export function AddProjectForm({ adding, error, onAdd, onClearError }: AddProjectFormProps) {
  const [expanded, setExpanded] = useState(false);
  const [remoteUrl, setRemoteUrl] = useState("");
  const [name, setName] = useState("");
  const [urlError, setUrlError] = useState<string | null>(null);

  const validate = useCallback((url: string): boolean => {
    if (!url.trim()) {
      setUrlError("Remote URL is required");
      return false;
    }
    if (!URL_PATTERN.test(url.trim())) {
      setUrlError("Must be a git remote URL (SSH or HTTPS)");
      return false;
    }
    setUrlError(null);
    return true;
  }, []);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!validate(remoteUrl)) return;

      const req: AddProjectRequest = { remote_url: remoteUrl.trim() };
      if (name.trim()) req.name = name.trim();

      const ok = await onAdd(req);
      if (ok) {
        setRemoteUrl("");
        setName("");
        setExpanded(false);
      }
    },
    [remoteUrl, name, onAdd, validate],
  );

  if (!expanded) {
    return (
      <button className={styles.addBtn} onClick={() => { setExpanded(true); onClearError(); }}>
        + Add Project
      </button>
    );
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit}>
      <div className={styles.fields}>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="remote-url">Remote URL</label>
          <input
            id="remote-url"
            className={styles.input}
            type="text"
            placeholder="git@github.com:user/repo.git"
            value={remoteUrl}
            onChange={(e) => { setRemoteUrl(e.target.value); setUrlError(null); onClearError(); }}
            autoFocus
            disabled={adding}
          />
          {urlError && <span className={styles.fieldError}>{urlError}</span>}
        </div>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="project-name">Name <span className={styles.optional}>(optional)</span></label>
          <input
            id="project-name"
            className={styles.input}
            type="text"
            placeholder="auto-derived from URL"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={adding}
          />
        </div>
      </div>
      {error && <div className={styles.error}>{error}</div>}
      <div className={styles.actions}>
        <button type="submit" className={styles.submitBtn} disabled={adding}>
          {adding ? "Adding..." : "Add Project"}
        </button>
        <button
          type="button"
          className={styles.cancelBtn}
          onClick={() => { setExpanded(false); setUrlError(null); onClearError(); }}
          disabled={adding}
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
