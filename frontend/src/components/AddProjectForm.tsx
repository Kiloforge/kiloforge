import { useState, useCallback, useEffect } from "react";
import type { AddProjectRequest } from "../types/api";
import { useSSHKeys } from "../hooks/useProjects";
import { useTourContextSafe } from "./tour/TourProvider";
import { TOUR_STEPS } from "./tour/tourSteps";
import styles from "./AddProjectForm.module.css";

interface AddProjectFormProps {
  adding: boolean;
  error: string | null;
  onAdd: (req: AddProjectRequest) => Promise<boolean>;
  onClearError: () => void;
}

const URL_PATTERN = /^(https?:\/\/.+|ssh:\/\/.+|[^/]+@[^:]+:.+)$/;
const SSH_URL_PATTERN = /^(ssh:\/\/.+|[^/]+@[^:]+:.+)$/;

export function AddProjectForm({ adding, error, onAdd, onClearError }: AddProjectFormProps) {
  const tour = useTourContextSafe();
  const [expanded, setExpanded] = useState(false);
  const [remoteUrl, setRemoteUrl] = useState("");
  const [name, setName] = useState("");
  const [sshKey, setSSHKey] = useState("");
  const [urlError, setUrlError] = useState<string | null>(null);
  const { keys: sshKeys, loading: keysLoading, fetchKeys } = useSSHKeys();

  const isSSH = SSH_URL_PATTERN.test(remoteUrl.trim());

  // Tour: auto-expand and prefill when on the add-project step
  const tourStep = tour?.isActive ? TOUR_STEPS[tour.currentStep] : null;
  useEffect(() => {
    if (tourStep?.id === "add-project" && !expanded) {
      setExpanded(true);
      setRemoteUrl("https://github.com/example/demo-app.git");
    }
  }, [tourStep?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  // Fetch SSH keys when an SSH URL is detected.
  useEffect(() => {
    if (isSSH) fetchKeys();
  }, [isSSH, fetchKeys]);

  // Auto-select when single key available.
  useEffect(() => {
    if (isSSH && sshKeys.length === 1 && !sshKey) {
      setSSHKey(sshKeys[0].path);
    }
  }, [isSSH, sshKeys, sshKey]);

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
      if (isSSH && sshKey) req.ssh_key = sshKey;

      const ok = await onAdd(req);
      if (ok) {
        setRemoteUrl("");
        setName("");
        setSSHKey("");
        setExpanded(false);
      }
    },
    [remoteUrl, name, sshKey, isSSH, onAdd, validate],
  );

  if (!expanded) {
    return (
      <button className={styles.addBtn} onClick={() => { setExpanded(true); onClearError(); }}>
        + Add Project
      </button>
    );
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit} data-tour="add-project-form">
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
      {isSSH && sshKeys.length > 0 && (
        <div className={styles.sshKeyField}>
          <label className={styles.label} htmlFor="ssh-key">SSH Key</label>
          <select
            id="ssh-key"
            className={styles.select}
            value={sshKey}
            onChange={(e) => setSSHKey(e.target.value)}
            disabled={adding || keysLoading}
          >
            <option value="">System default</option>
            {sshKeys.map((k) => (
              <option key={k.path} value={k.path}>
                {k.name} ({k.type}){k.comment ? ` — ${k.comment}` : ""}
              </option>
            ))}
          </select>
        </div>
      )}
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
