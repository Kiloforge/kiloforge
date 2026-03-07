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

## `crelay destroy`

Tear down the Gitea Docker Compose stack.

**Synopsis:**
```bash
crelay destroy [--data]
```

**What it does:**
1. Runs `docker compose down` to stop and remove containers
2. With `--data`: also runs with `--volumes` and removes the data directory

**Flags:**

| Flag | Description |
|------|-------------|
| `--data` | Also delete `~/.crelay/` (config, state, logs, Gitea volumes) |

---

## Coming Soon

The following commands are temporarily disabled and will return with the `crelay add` feature:

- **`crelay add`** — Register a project with the global Gitea server
- **`crelay agents`** — List active and recent agents
- **`crelay logs <agent-id>`** — View agent log output
- **`crelay attach <agent-id>`** — Resume an agent interactively
- **`crelay stop <agent-id>`** — Stop a running agent
