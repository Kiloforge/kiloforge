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

# Register your project
cd ~/dev/my-project
crelay add .

# List registered projects
crelay projects
```

This will:
1. Detect your Docker Compose CLI variant (v2 or v1)
2. Generate a `docker-compose.yml` in `~/.crelay/`
3. Start a Gitea instance at `http://localhost:3000`
4. Create an admin user (`conductor` / `conductor123`)
5. Generate an API token and save config
6. Register your project: create Gitea repo, add remote, push code

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

Start Gitea and the webhook relay server (daily use). The relay runs in the foreground — press Ctrl+C to stop it. Gitea stays running via Docker Compose.

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

### `crelay add`

Register a project with the Gitea server.

```bash
crelay add [repo-path]    # defaults to current directory
crelay add . --name myapp # override project slug
crelay add . --origin git@github.com:user/repo.git  # override origin
```

Creates a Gitea repo, adds a `gitea` remote, pushes the main branch, and registers a webhook. The project's origin remote URL is captured for future bridging support.

### `crelay projects`

List registered projects.

```bash
crelay projects
```

### `crelay destroy`

Permanently destroy all crelay data (requires confirmation).

```bash
crelay destroy          # prompts for confirmation
crelay destroy --force  # skip confirmation
```

## Architecture

```
crelay init / crelay up
    │
    ├─ Docker Compose: start Gitea (localhost:3000)
    ├─ Webhook relay server (localhost:3001, foreground)
    │   ├─ Receives events from all registered projects
    │   ├─ Routes by repository name → project registry
    │   └─ Handles: issues, issue_comment, pull_request,
    │              pull_request_review, pull_request_comment, push
    │
    └─ crelay add: register project → Gitea repo + webhook

┌─────────────────────────────────────────────────────────────┐
│  Gitea (Docker Compose)                    localhost:3000    │
│  • Hosts git repos for multiple projects                    │
│  • Manages PRs and reviews                                  │
│  • Sends webhooks to relay on events                        │
└────────────────────────┬────────────────────────────────────┘
                         │ webhooks
┌────────────────────────▼────────────────────────────────────┐
│  Relay Server                              localhost:3001    │
│  • Multi-project event routing                              │
│  • Structured logging per project                           │
│  • Health: GET /health                                      │
└─────────────────────────────────────────────────────────────┘
```

## Supported Events

| Event | Actions |
|-------|---------|
| `issues` | opened, edited, closed, label_updated, assigned |
| `issue_comment` | created |
| `pull_request` | opened, reopened, closed, merged, synchronize |
| `pull_request_review` | submitted |
| `pull_request_comment` | created |
| `push` | (all) |

## Data Directory

All persistent data lives in `~/.crelay/` (configurable via `--data-dir`):

```
~/.crelay/
├── config.json           # Global configuration
├── projects.json         # Project registry
├── docker-compose.yml    # Generated compose file
├── projects/             # Per-project data
│   └── <slug>/
│       └── logs/
└── gitea-data/           # Gitea Docker volume (repos, DB)
```

## Origin Bridging

When you register a project with `crelay add`, the origin remote URL is captured and stored. This enables a future workflow: develop locally against Gitea (PRs, reviews, CI), then bridge changes back to your real remote (GitHub, GitLab) with a single command.

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
