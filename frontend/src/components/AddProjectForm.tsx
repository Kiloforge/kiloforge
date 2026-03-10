import { useState, useCallback, useEffect } from "react";
import type { AddProjectRequest } from "../types/api";
import { useSSHKeys } from "../hooks/useProjects";
import styles from "./AddProjectForm.module.css";

type FormMode = "clone" | "create";

interface AddProjectFormProps {
  adding: boolean;
  error: string | null;
  onAdd: (req: AddProjectRequest) => Promise<boolean>;
  onClearError: () => void;
}

const URL_PATTERN = /^(https?:\/\/.+|ssh:\/\/.+|[^/]+@[^:]+:.+)$/;
const SSH_URL_PATTERN = /^(ssh:\/\/.+|[^/]+@[^:]+:.+)$/;

export function AddProjectForm({ adding, error, onAdd, onClearError }: AddProjectFormProps) {
  const [expanded, setExpanded] = useState(false);
  const [mode, setMode] = useState<FormMode>("clone");
  const [remoteUrl, setRemoteUrl] = useState("");
  const [name, setName] = useState("");
  const [sshKey, setSSHKey] = useState("");
  const [urlError, setUrlError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);
  const { keys: sshKeys, loading: keysLoading, fetchKeys } = useSSHKeys();

  const isSSH = mode === "clone" && SSH_URL_PATTERN.test(remoteUrl.trim());

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

  const validate = useCallback((): boolean => {
    if (mode === "clone") {
      if (!remoteUrl.trim()) {
        setUrlError("Remote URL is required");
        return false;
      }
      if (!URL_PATTERN.test(remoteUrl.trim())) {
        setUrlError("Must be a git remote URL (SSH or HTTPS)");
        return false;
      }
      setUrlError(null);
      return true;
    }
    // create mode
    if (!name.trim()) {
      setNameError("Project name is required");
      return false;
    }
    setNameError(null);
    return true;
  }, [mode, remoteUrl, name]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!validate()) return;

      let req: AddProjectRequest;
      if (mode === "clone") {
        req = { remote_url: remoteUrl.trim() };
        if (name.trim()) req.name = name.trim();
        if (isSSH && sshKey) req.ssh_key = sshKey;
      } else {
        req = { name: name.trim() };
      }

      const ok = await onAdd(req);
      if (ok) {
        setRemoteUrl("");
        setName("");
        setSSHKey("");
        setExpanded(false);
        setMode("clone");
      }
    },
    [mode, remoteUrl, name, sshKey, isSSH, onAdd, validate],
  );

  const handleModeChange = useCallback((newMode: FormMode) => {
    setMode(newMode);
    setUrlError(null);
    setNameError(null);
  }, []);

  if (!expanded) {
    return (
      <button className={styles.addBtn} onClick={() => { setExpanded(true); onClearError(); }}>
        + Add Project
      </button>
    );
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit} data-tour="add-project-form">
      <div className={styles.modeToggle}>
        <button
          type="button"
          className={`${styles.modeBtn} ${mode === "clone" ? styles.modeBtnActive : ""}`}
          onClick={() => handleModeChange("clone")}
          disabled={adding}
        >
          Clone from remote
        </button>
        <button
          type="button"
          className={`${styles.modeBtn} ${mode === "create" ? styles.modeBtnActive : ""}`}
          onClick={() => handleModeChange("create")}
          disabled={adding}
        >
          Create new
        </button>
      </div>

      <div className={styles.fields}>
        {mode === "clone" && (
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
        )}
        <div className={styles.field}>
          <label className={styles.label} htmlFor="project-name">
            Name {mode === "clone" && <span className={styles.optional}>(optional)</span>}
          </label>
          <input
            id="project-name"
            className={styles.input}
            type="text"
            placeholder={mode === "clone" ? "auto-derived from URL" : "my-project"}
            value={name}
            onChange={(e) => { setName(e.target.value); setNameError(null); }}
            autoFocus={mode === "create"}
            disabled={adding}
          />
          {nameError && <span className={styles.fieldError}>{nameError}</span>}
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
          {adding ? "Adding..." : mode === "clone" ? "Clone Project" : "Create Project"}
        </button>
        <button
          type="button"
          className={styles.cancelBtn}
          onClick={() => { setExpanded(false); setUrlError(null); setNameError(null); onClearError(); }}
          disabled={adding}
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
