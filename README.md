# Conductor Relay

A local Gitea instance managed via Docker Compose that serves as the git forge for [Conductor](https://github.com/your-org/ai-skills) workflows with Claude Code agents.

## Why

Conductor's role-based agents (developer, reviewer) work best with a git forge for PRs and code review. Running Gitea locally gives you:

- **Free, private, fast** — no GitHub rate limits or network latency
- **Automatic agent orchestration** — webhooks trigger Claude agents (reviewer spawns when a PR is opened)
- **Session management** — view logs, halt agents, and resume their Claude sessions interactively
- **Full control** — everything runs on your machine

## Prerequisites

- **Docker** with Docker Compose — either Docker Desktop (includes compose v2) or Docker Engine + `docker-compose` (v1, for Colima users)
- **Claude Code CLI** — `claude` command available in PATH
- **Go 1.21+** — to build (or use prebuilt binary)

### Colima Users

If you're using Colima on macOS, install docker-compose separately:

```bash
brew install docker-compose
```

Both `docker compose` (v2) and `docker-compose` (v1) are auto-detected.

## Quick Start

```bash
# Build
make build

# Initialize the global Gitea server
crelay init

# That's it. Gitea is running via Docker Compose.
```

This will:
1. Detect your Docker Compose CLI variant (v2 or v1)
2. Generate a `docker-compose.yml` in `~/.crelay/`
3. Start a Gitea instance at `http://localhost:3000`
4. Create an admin user (`conductor` / `conductor123`)
5. Generate an API token and save config

## Commands

### `crelay init`

One-time setup: start the global Gitea server via Docker Compose.

```bash
crelay init [flags]

Flags:
  --gitea-port int   Port for Gitea web UI (default 3000)
  --data-dir string  Persistent data directory (default ~/.crelay)
```

**Idempotent:** Running again when Gitea is already running prints the status and exits.

### `crelay up`

Start the Gitea server (daily use).

```bash
crelay up
```

### `crelay down`

Stop the Gitea server without removing data (daily use).

```bash
crelay down
```

### `crelay status`

Show Gitea server status.

```bash
$ crelay status
Conductor Relay Status
======================
Gitea:       running (v1.22.0) — http://localhost:3000
Data:        /Users/you/.crelay
Compose:     /Users/you/.crelay/docker-compose.yml
```

### `crelay destroy`

Permanently destroy all crelay data (requires confirmation).

```bash
crelay destroy          # prompts for confirmation
crelay destroy --force  # skip confirmation
```

## Architecture

```
crelay init
    │
    ├─ Detect compose CLI (v2 plugin or v1 standalone)
    ├─ Generate docker-compose.yml in ~/.crelay/
    ├─ docker compose up -d (Gitea service)
    ├─ Wait for Gitea health check
    ├─ Create admin user via compose exec
    ├─ Create API token via REST API
    └─ Save global config

┌──────────────────────────────────────────────────────────┐
│  Gitea (Docker Compose)                 localhost:3000    │
│                                                          │
│  • Hosts git repos for multiple projects                 │
│  • Manages PRs and reviews                               │
│  • Sends webhooks on events                              │
│  • Persistent data in ~/.crelay/gitea-data               │
└──────────────────────────────────────────────────────────┘
```

## Data Directory

All persistent data lives in `~/.crelay/` (configurable via `--data-dir`):

```
~/.crelay/
├── config.json           # Global configuration
├── docker-compose.yml    # Generated compose file
├── state.json            # Agent tracking state
├── logs/                 # Agent log files
│   └── <agent-id>.log
└── gitea-data/           # Gitea Docker volume (repos, DB)
```

## What's Next

Project registration (`crelay add`) is coming in a future release. This will allow you to:
- Register projects with the global Gitea instance
- Set up git remotes and webhooks per project
- Start the relay server for webhook-driven agent orchestration

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint
```

## License

MIT
