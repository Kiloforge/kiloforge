# Architecture

## Overview

Kiloforge manages a global Gitea instance via Docker Compose, providing a local git forge for Conductor-based development and automated code review with Claude Code agents.

## Components

### 1. CLI (`internal/cli/`)

Cobra-based command-line interface:

- **`init.go`** — Detects compose CLI, generates compose file, starts Gitea, configures admin
- **`status.go`** — Checks Gitea API health and shows compose container status
- **`destroy.go`** — Tears down via `docker compose down`

Project-specific commands (`agents`, `logs`, `attach`, `stop`) are disabled until project registration (`kf add`) is implemented.

### 2. Compose Runner (`internal/compose/`)

Abstracts Docker Compose CLI differences:

- **`runner.go`** — Detects v2 (`docker compose`) vs v1 (`docker-compose`), provides `Up`, `Down`, `Ps`, `Exec` methods
- **`template.go`** — Generates `docker-compose.yml` with Gitea service, healthcheck, and Colima-compatible `extra_hosts`

### 3. Gitea Manager (`internal/gitea/manager.go`)

Handles Gitea lifecycle through the compose runner:

- **Start**: Runs `compose up -d` then waits for API readiness
- **WaitReady**: Polls Gitea API until it responds (up to 60s)
- **Configure**: Creates admin user via `compose exec`, creates API token via REST API

### 4. Gitea Client (`internal/gitea/client.go`)

Thin wrapper around Gitea's REST API:

- Authentication (token or basic auth)
- Token creation, repo creation, webhook registration
- PR fetching, version checking

### 5. Config (`internal/config/config.go`)

Global configuration stored at `~/.kiloforge/config.json`:

```json
{
  "gitea_port": 3000,
  "data_dir": "/Users/you/.kiloforge",
  "api_token": "...",
  "compose_file": "/Users/you/.kiloforge/docker-compose.yml"
}
```

### 6. Agent Spawner (`internal/agent/spawner.go`)

Manages Claude Code process lifecycle (currently used by orchestrator, disabled in CLI):

- **SpawnReviewer**: Launches `claude -p "/kf-reviewer <pr-url>"`
- **SpawnDeveloper**: Launches `claude -p "/kf-developer <track> <flags>"`

### 7. State Store (`internal/state/state.go`)

JSON file-based agent state persistence at `~/.kiloforge/state.json`.

### 8. Orchestrator (`internal/rest/server.go`)

HTTP server for webhook handling, agent orchestration, dashboard, and lock service.

## Data Flow

```
kf init
    │
    ├─ compose.Detect() → finds docker compose v2 or v1
    ├─ compose.GenerateComposeFile() → renders docker-compose.yml
    ├─ runner.Up() → docker compose up -d
    ├─ manager.waitReady() → polls /api/v1/version
    ├─ runner.Exec() → gitea admin user create
    ├─ client.CreateToken() → POST /api/v1/users/conductor/tokens
    └─ config.Save() → ~/.kiloforge/config.json
```

## Docker Compose Setup

The generated `docker-compose.yml` defines:

- **Gitea service** — `gitea/gitea:latest` with SQLite, registration disabled
- **Port mapping** — configurable host port → container port 3000
- **Volume** — bind mount at `~/.kiloforge/gitea-data:/data`
- **Health check** — polls `/api/v1/version` every 5s
- **Colima support** — `extra_hosts: host.docker.internal:host-gateway`

## Future: Project Registration

The `kf add` command will:
1. Create a repository in the global Gitea instance
2. Add a `gitea` git remote to the project
3. Push the project to Gitea
4. Register webhooks
5. Re-enable orchestrator and agent management commands
