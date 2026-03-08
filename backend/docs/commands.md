# Command Reference

## `kf init`

Initialize the global Gitea server via Docker Compose.

**Synopsis:**
```bash
kf init [--gitea-port PORT] [--data-dir PATH]
```

**What it does (in order):**
1. Detects Docker Compose CLI variant (v2 plugin or v1 standalone)
2. Creates data directory (`~/.kiloforge/`)
3. Generates `docker-compose.yml` with Gitea service definition
4. Runs `docker compose up -d` to start Gitea
5. Waits for Gitea to become healthy (up to 60s)
6. Creates admin user `conductor` via `docker compose exec`
7. Creates API access token via REST API
8. Saves configuration to `~/.kiloforge/config.json`

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--gitea-port` | `3000` | Port for Gitea web UI and API |
| `--data-dir` | `~/.kiloforge` | Where to store config, compose file, and Gitea data |

**Idempotent:** If Gitea is already running, reports the status and exits without making changes.

---

## `kf status`

Display the current status of the Gitea server.

**Synopsis:**
```bash
kf status
```

**Output:**
```
Kiloforge Status
======================
Gitea:       running (v1.22.0) — http://localhost:3000
Data:        /Users/you/.kiloforge
Compose:     /Users/you/.kiloforge/docker-compose.yml
```

**Checks performed:**
- Gitea API version endpoint for liveness and version
- Docker Compose `ps` for container details

---

## `kf up`

Start Gitea and the orchestrator (daily use).

**Synopsis:**
```bash
kf up
```

**What it does:**
1. Loads saved config — errors if not initialized
2. Starts Gitea via Docker Compose (if not already running)
3. Starts the orchestrator as a background daemon on the configured port (default 3001)

**Requires:** `kf init` must have been run first.

**Orchestrator behavior:**
- Routes webhook events from Gitea to the correct project via `repository.name`
- Handles: issues, issue_comment, pull_request, pull_request_review, pull_request_comment, push
- Logs structured output per project: `[orchestrator] [project-slug] event: details`

---

## `kf down`

Stop the Gitea server without removing data (daily use).

**Synopsis:**
```bash
kf down
```

**What it does:**
1. Loads saved config — errors if not initialized
2. Checks if Gitea is running (no-op if already stopped)
3. Runs `docker compose stop` to stop containers without removing them

**Non-destructive:** Containers and data are preserved. Restart with `kf up`.

---

## `kf destroy`

Permanently destroy all kiloforge data.

**Synopsis:**
```bash
kf destroy [--force]
```

**What it does:**
1. Prints a critical warning listing everything that will be deleted
2. Requires typing "yes" to confirm (use `--force` to skip)
3. Runs `docker compose down --volumes` to remove containers and volumes
4. Removes the entire data directory (`~/.kiloforge/`)

**Flags:**

| Flag | Description |
|------|-------------|
| `--force` | Skip the confirmation prompt |

**Example:**
```
$ kf destroy

  WARNING: This will permanently delete:
    - Gitea server and all repositories
    - All project registrations
    - All agent state and logs
    - Data directory: /Users/you/.kiloforge

  This action cannot be undone.

  Type "yes" to confirm:
```

---

## `kf add`

Clone a remote repo and register it with the Gitea server.

**Synopsis:**
```bash
kf add <remote-url> [--name SLUG]
```

**What it does (in order):**
1. Validates the argument is a remote URL (SSH or HTTPS)
2. Derives a project slug from the URL (last path component, minus `.git`)
3. Loads global config and verifies Gitea is running
4. Clones the remote into `~/.kiloforge/repos/<slug>/`
5. Creates a repository in Gitea via API
6. Adds a `gitea` git remote to the cloned repo
7. Pushes the main branch to Gitea
8. Registers a webhook for orchestrator events
9. Creates project data directory (`~/.kiloforge/projects/<slug>/logs/`)
10. Saves the project to `~/.kiloforge/projects.json`

**Flags:**

| Flag | Description |
|------|-------------|
| `--name` | Override the project slug (defaults to repo name from URL) |

**Idempotent:** Re-adding an already-registered project prints the existing registration and exits.

**Example:**
```
$ kf add git@github.com:you/my-project.git
==> Cloning git@github.com:you/my-project.git...
==> Creating Gitea repo 'my-project'...
==> Adding gitea remote...
    Remote: http://localhost:3000/conductor/my-project.git
==> Pushing to Gitea...
==> Registering webhook...

Project 'my-project' registered!
  Path:   /Users/you/.kiloforge/repos/my-project
  Gitea:  http://localhost:3000/conductor/my-project
  Origin: git@github.com:you/my-project.git

View registered projects with 'kf projects'.
```

---

## `kf projects`

List all registered projects.

**Synopsis:**
```bash
kf projects
```

**Output:**
```
SLUG        PATH                      ORIGIN                              REGISTERED  ACTIVE
my-project  /Users/you/dev/my-project git@github.com:you/my-project.git   2026-03-07  yes
```

---

## `kf pool`

Show worktree pool status.

**Synopsis:**
```bash
kf pool
```

**Output:**
```
Worktree Pool (2/3)

NAME       STATUS  TRACK           AGENT      ACQUIRED
worker-1   idle    -               -          -
worker-2   in-use  auth_20260307   uuid-123   2026-03-07 12:00:00
```

Worktrees are created automatically by `kf implement` when needed. The pool manages reusable git worktrees for developer agents, avoiding the overhead of creating and destroying worktrees per task.

---

## `kf escalated`

Show PRs that hit the review cycle limit and require human intervention.

**Synopsis:**
```bash
kf escalated
```

**Output:**
```
Escalated PRs (1)

PROJECT  PR#  TRACK              CYCLES
myapp    #5   auth_20260307...   3
```

When a PR exceeds the maximum review cycle count (default 3), the orchestrator labels the PR `needs-human-review`, posts a comment, and stops all agents. Use this command to find such PRs.

---

## Review Cycle

The orchestrator manages the developer-reviewer cycle automatically:

1. **PR opened** → reviewer agent spawned
2. **Review approved** → developer resumed for merge
3. **Changes requested** → developer resumed for revisions (cycle count incremented)
4. **Developer pushes** → new reviewer spawned for re-review
5. **Cycle limit reached** → PR escalated, agents stopped

The default max review cycles is 3.
