# Specification: Implement 'crelay add' Command for Project Registration

**Track ID:** add-project-command_20260307122000Z
**Type:** Feature
**Created:** 2026-03-07T12:20:00Z
**Status:** Draft

## Summary

Implement `crelay add <repo-path>` to register a project with the global Gitea instance. Records the project's origin remote URL in the project registry so crelay can bridge local Gitea work back to the real remote on demand. Creates a Gitea repo, adds a `gitea` git remote, pushes code, registers a webhook, and tracks the project in a new `projects.json` registry. Also adds `crelay projects` to list registered projects.

## Context

After `crelay init`, the user has a running Gitea instance but no projects registered. The `add` command bridges a local repo into the crelay system. The key architectural insight is that crelay works locally against Gitea but needs to push changes back to the user's real remote (e.g., GitHub) when asked. To support this, the project registry stores the origin remote URL captured at registration time.

The research doc (`docs/research-global-gitea-multiproject.md`) defines the project registry schema, onboarding flow, and directory structure. This track implements those designs.

## Codebase Analysis

- **`internal/cli/root.go`** — Currently registers `init`, `status`, `destroy`. Project-specific commands (agents, logs, attach, stop) are commented out pending project context. `add` and `projects` commands need to be registered here.
- **`internal/gitea/client.go`** — Already has `CreateRepo()`, `CreateWebhook()`, `CheckVersion()` methods. These are used directly by `add`.
- **`internal/gitea/manager.go`** — `Configure()` returns a `*Client` with token. The `add` command needs a client — it can construct one from saved config (API token).
- **`internal/config/config.go`** — Global config with `GiteaPort`, `DataDir`, `APIToken`, `ComposeFile`. No project registry yet. Constants `GiteaAdminUser`, `GiteaAdminPass` used for client auth.
- **`internal/cli/init.go`** — Line 123: `fmt.Println("Next: use 'crelay add' to register a project (coming soon).")` — this placeholder gets fulfilled.
- **`internal/state/state.go`** — Agent state store. Currently loads from `{dataDir}/state.json`. For multi-project, should load from `{dataDir}/projects/{slug}/state.json`.
- **`internal/relay/server.go`** — `NewServer()` takes `repoName` as explicit param. Multi-project routing is a future concern — this track focuses on registration only.

## Acceptance Criteria

- [ ] `crelay add <repo-path>` registers a project with the global Gitea instance
- [ ] `crelay add` with no args uses the current working directory
- [ ] The project's origin remote URL is captured and stored in the registry (reads from `git remote get-url origin` in the project dir)
- [ ] `--name` flag allows overriding the project slug (defaults to directory basename)
- [ ] `--origin` flag allows overriding the origin remote URL (defaults to auto-detected)
- [ ] Project registry (`projects.json`) created/updated in `~/.crelay/`
- [ ] Gitea repo created via API
- [ ] `gitea` git remote added to the project pointing to local Gitea
- [ ] Main branch pushed to Gitea
- [ ] Webhook registered on the Gitea repo
- [ ] Project data directory created: `~/.crelay/projects/<slug>/logs/`
- [ ] `crelay projects` lists all registered projects with slug, path, origin, and status
- [ ] Idempotent: running `add` on an already-registered project prints a message and exits cleanly
- [ ] Error if Gitea is not running (directs user to run `crelay init` first)
- [ ] Error if the target path is not a git repository

## Dependencies

None (init-docker-compose track is already complete)

## Out of Scope

- `crelay remove` command (future track)
- `crelay push` / bridge command to push back to origin (future track — registry stores origin URL to enable this)
- Relay webhook routing for multi-project (future track)
- Re-enabling project-scoped agent commands (agents, logs, attach, stop) with project context resolution
- SQLite migration for project state
- Gitea organizations / namespace isolation

## Technical Notes

### Project Registry Schema (`projects.json`)

```json
{
  "version": 1,
  "projects": {
    "crelay": {
      "slug": "crelay",
      "repo_name": "crelay",
      "project_dir": "/Users/dev/crelay",
      "origin_remote": "git@github.com:user/crelay.git",
      "registered_at": "2026-03-07T12:00:00Z",
      "active": true
    }
  }
}
```

The `origin_remote` field stores the URL of the project's `origin` git remote at the time of registration. This enables future bridging: work locally against Gitea, then `crelay push` syncs back to the real remote.

### Add Flow

```
1. Parse <repo-path> arg (default: cwd)
2. Verify path is a git repo (check for .git)
3. Load global config (verify Gitea is initialized)
4. Verify Gitea is running (CheckVersion API call)
5. Derive slug from basename (or --name flag)
6. Check projects.json — error if slug already registered
7. Detect origin remote: git -C <path> remote get-url origin
8. Allow --origin override
9. Create Gitea client from saved config (APIToken or basic auth)
10. Create repo in Gitea: client.CreateRepo(slug)
11. Add gitea remote: git -C <path> remote add gitea http://.../<slug>.git
12. Push main: git -C <path> push -u gitea main
13. Create webhook: client.CreateWebhook(slug, relayPort)
14. Create project data dir: ~/.crelay/projects/<slug>/logs/
15. Register in projects.json
16. Print success with Gitea repo URL
```

### Origin Remote Capture

```go
// Detect origin remote URL
out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", "origin").Output()
if err != nil {
    // No origin remote — warn but continue (origin_remote will be empty)
    fmt.Println("    Warning: no 'origin' remote found — origin bridging won't be available")
}
originRemote := strings.TrimSpace(string(out))
```

If `--origin` is provided, it overrides the auto-detected value. If there's no origin remote and no `--origin` flag, the field is left empty with a warning.

---

_Generated by conductor-track-generator from prompt: "Add a track for adding a repo to the system via 'crelay add' with origin remote tracking for bridging"_
