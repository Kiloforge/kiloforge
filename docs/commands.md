# Command Reference

## `crelay init`

Initialize the global Gitea server via Docker Compose.

**Synopsis:**
```bash
crelay init [--gitea-port PORT] [--data-dir PATH]
```

**What it does (in order):**
1. Detects Docker Compose CLI variant (v2 plugin or v1 standalone)
2. Creates data directory (`~/.crelay/`)
3. Generates `docker-compose.yml` with Gitea service definition
4. Runs `docker compose up -d` to start Gitea
5. Waits for Gitea to become healthy (up to 60s)
6. Creates admin user `conductor` via `docker compose exec`
7. Creates API access token via REST API
8. Saves configuration to `~/.crelay/config.json`

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--gitea-port` | `3000` | Port for Gitea web UI and API |
| `--data-dir` | `~/.crelay` | Where to store config, compose file, and Gitea data |

**Idempotent:** If Gitea is already running, reports the status and exits without making changes.

---

## `crelay status`

Display the current status of the Gitea server.

**Synopsis:**
```bash
crelay status
```

**Output:**
```
Conductor Relay Status
======================
Gitea:       running (v1.22.0) — http://localhost:3000
Data:        /Users/you/.crelay
Compose:     /Users/you/.crelay/docker-compose.yml
```

**Checks performed:**
- Gitea API version endpoint for liveness and version
- Docker Compose `ps` for container details

---

## `crelay up`

Start the Gitea server (daily use).

**Synopsis:**
```bash
crelay up
```

**What it does:**
1. Loads saved config — errors if not initialized
2. Checks if Gitea is already running (no-op if so)
3. Runs `docker compose up -d` to start the stack
4. Waits for Gitea to become healthy
5. Prints the Gitea URL

**Requires:** `crelay init` must have been run first.

---

## `crelay down`

Stop the Gitea server without removing data (daily use).

**Synopsis:**
```bash
crelay down
```

**What it does:**
1. Loads saved config — errors if not initialized
2. Checks if Gitea is running (no-op if already stopped)
3. Runs `docker compose stop` to stop containers without removing them

**Non-destructive:** Containers and data are preserved. Restart with `crelay up`.

---

## `crelay destroy`

Permanently destroy all crelay data.

**Synopsis:**
```bash
crelay destroy [--force]
```

**What it does:**
1. Prints a critical warning listing everything that will be deleted
2. Requires typing "yes" to confirm (use `--force` to skip)
3. Runs `docker compose down --volumes` to remove containers and volumes
4. Removes the entire data directory (`~/.crelay/`)

**Flags:**

| Flag | Description |
|------|-------------|
| `--force` | Skip the confirmation prompt |

**Example:**
```
$ crelay destroy

  WARNING: This will permanently delete:
    - Gitea server and all repositories
    - All project registrations
    - All agent state and logs
    - Data directory: /Users/you/.crelay

  This action cannot be undone.

  Type "yes" to confirm:
```

---

## Coming Soon

The following commands are planned for future releases:

- **`crelay add`** — Register a project with the global Gitea server
- **`crelay agents`** — List active and recent agents
- **`crelay logs <agent-id>`** — View agent log output
- **`crelay attach <agent-id>`** — Resume an agent interactively
- **`crelay stop <agent-id>`** — Stop a running agent
