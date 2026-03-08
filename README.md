# Kiloforge

**1,000x Productivity.** Forge code at the speed of thought with AI agent swarms.

An orchestration platform for coordinating AI coding agents at scale. Runs a private git forge (Gitea), a real-time monitoring dashboard, and a relay server on your machine — while spawning Claude Code CLI agents that implement, review, and merge code autonomously.

## Why

Coordinating multiple AI agents across multiple projects demands infrastructure that is observable, automated, and under your control. Kiloforge gives you:

- **Private infrastructure, cloud AI** — git forge, webhooks, and coordination run locally; agents are Claude Code CLI sessions powered by Anthropic's cloud APIs
- **Human + AI collaboration** — Gitea for code review and PRs, plus a web dashboard for real-time agent monitoring, quota tracking, and log streaming
- **Agent orchestration at scale** — spawn, monitor, throttle, suspend, and resume dozens of concurrent agents across multiple projects
- **Session persistence** — gracefully shut down agents and auto-recover them on restart, with full session continuity
- **Quota-aware** — track token usage and cost per agent/track, enforce budgets, and handle rate limits gracefully
- **End-to-end tracing** — OpenTelemetry traces follow each track from claim through agent work, PR review, and merge
- **Extensible** — scoped lock service, webhook relay, and REST APIs that agents and tools can build on
- **Full control** — your code stays on your machine; only requires Git, Docker, and Claude Code

## Prerequisites

- **Git** — `git` command available in PATH
- **Docker** with Docker Compose — either Docker Desktop (includes compose v2) or Docker Engine + `docker-compose` (v1, for Colima users)
- **Claude Code CLI** — `claude` command available in PATH

### Building from Source

- **Go 1.25+**
- **Node.js 18+**

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
kf init

# Register your project
kf add git@github.com:user/my-project.git

# List registered projects
kf projects
```

This will:
1. Detect your Docker Compose CLI variant (v2 or v1)
2. Generate a `docker-compose.yml` in `~/.kiloforge/`
3. Start a Gitea instance at `http://localhost:3000`
4. Create an admin user (`conductor` / random password)
5. Generate an API token and save config
6. Register your project: create Gitea repo, add remote, push code

## Commands

### `kf init`

One-time setup: start the global Gitea server via Docker Compose.

```bash
kf init [flags]

Flags:
  --gitea-port int    Port for Gitea web UI (default 3000)
  --data-dir string   Persistent data directory (default ~/.kiloforge)
  --admin-pass string Admin password (default: generated random)
  --ssh-key string    Path to SSH public key (default: auto-detect)
```

On first init, a random admin password is generated and saved to `config.json`. Subsequent runs reuse the saved password. Use `--admin-pass` to override.

Your SSH public key is auto-detected from `~/.ssh/` (tries `id_ed25519.pub`, `id_rsa.pub`, `id_ecdsa.pub`) and registered with the Gitea admin user. Use `--ssh-key` to specify a custom path. Missing SSH keys produce a warning but do not prevent initialization.

**Idempotent:** Running again when Gitea is already running prints the status and exits.

### `kf up`

Start Gitea and the orchestrator (daily use). Returns immediately after both are running.

```bash
kf up
```

### `kf down`

Stop the Gitea server without removing data (daily use).

```bash
kf down
```

### `kf status`

Show Gitea server status.

```bash
$ kf status
Kiloforge Status
======================
Gitea:       running (v1.22.0) — http://localhost:3000
Data:        /Users/you/.kiloforge
Compose:     /Users/you/.kiloforge/docker-compose.yml
```

### `kf add`

Clone a remote repo and register it with the Gitea server.

```bash
kf add git@github.com:user/repo.git          # SSH URL
kf add https://github.com/user/repo.git      # HTTPS URL
kf add git@github.com:user/repo.git --name x  # override slug
```

Clones the remote into `~/.kiloforge/repos/<slug>/`, creates a Gitea repo, adds a `gitea` remote, pushes the main branch, and registers a webhook.

### `kf projects`

List registered projects.

```bash
kf projects
```

### `kf implement`

Approve a conductor track and spawn a developer agent in a pooled worktree.

```bash
kf implement <track-id>            # spawn developer for track
kf implement --list                # list available tracks
kf implement --project myapp <id>  # specify project explicitly
```

The command acquires a worktree from the pool, prepares it (reset to main, create implementation branch), and spawns a Claude Code agent running `/conductor-developer <track-id>`. Agent state is recorded for monitoring with `kf agents`, `kf logs`, `kf stop`, and `kf attach`.

### `kf agents`

List active and recent agents.

```bash
kf agents          # table output
kf agents --json   # JSON output
```

### `kf logs <agent-id>`

View logs for an agent. Supports prefix matching on the agent ID.

```bash
kf logs abc12345
kf logs abc12345 -f   # follow mode
```

### `kf stop <agent-id>`

Send SIGINT to stop a running agent. The session is preserved for later resume.

### `kf attach <agent-id>`

Print the command to resume an agent's Claude session interactively. If the agent is running, it is halted first.

### `kf pool`

Show worktree pool status. Displays idle and in-use worktrees for developer agents.

```bash
kf pool
```

### `kf escalated`

Show PRs that hit the review cycle limit and require human intervention.

```bash
kf escalated
```

### `kf destroy`

Permanently destroy all kiloforge data (requires confirmation).

```bash
kf destroy          # prompts for confirmation
kf destroy --force  # skip confirmation
```

## Architecture

```
kf init / kf up
    │
    ├─ Docker Compose: start Gitea (localhost:3000)
    ├─ Orchestrator (localhost:3001)
    │   ├─ Receives events from all registered projects
    │   ├─ Routes by repository name → project registry
    │   ├─ Scoped lock service (merge coordination)
    │   └─ Handles: issues, PRs, reviews, push events
    │
    ├─ Dashboard (localhost:3002)
    │   ├─ Real-time agent status via SSE
    │   ├─ Quota/cost monitoring
    │   └─ Log streaming
    │
    └─ kf add: register project → Gitea repo + webhook

┌─────────────────────────────────────────────────────────────┐
│  Gitea (Docker)                            localhost:3000    │
│  • Git repos, PRs, code review for multiple projects        │
│  • Webhooks → orchestrator on events                         │
└────────────────────────┬────────────────────────────────────┘
                         │ webhooks
┌────────────────────────▼────────────────────────────────────┐
│  Orchestrator                              localhost:3001    │
│  • Multi-project event routing                              │
│  • Developer-reviewer review cycle                          │
│  • Agent lifecycle: spawn, suspend, resume                  │
│  • Quota tracking and budget enforcement                    │
│  • Scoped lock API (merge serialization)                    │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Dashboard                                 localhost:3002    │
│  • Agent status, logs, and cost — live in the browser       │
│  • Links to Gitea PRs and repos                             │
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

All persistent data lives in `~/.kiloforge/` (configurable via `--data-dir`):

```
~/.kiloforge/
├── config.json           # Global configuration
├── projects.json         # Project registry
├── pool.json             # Worktree pool state
├── state.json            # Agent state (running/completed agents)
├── docker-compose.yml    # Generated compose file
├── repos/                # Cloned project repositories
│   └── <slug>/
├── projects/             # Per-project data
│   └── <slug>/
│       ├── logs/             # Agent log files
│       └── pr-tracking.json  # PR-to-agent tracking
└── gitea-data/           # Gitea Docker volume (repos, DB)
```

## Tracing

Kiloforge supports OpenTelemetry distributed tracing with **track lifecycle tracing** — a single trace follows a development track from claim through agent work, PR review, merge, and completion. This gives end-to-end visibility into the full lifecycle of every track.

When enabled:
- **`kf implement`** creates a root span `track/{trackId}` with child spans for worktree acquisition, agent spawning, and session tracking
- **Webhook events** (PR opened, review submitted, merge) automatically join the track's trace via stored trace IDs, so all activity for a track appears in one trace
- **Agent spans** include `session.id` attributes for cross-referencing with Claude Code sessions
- **The dashboard** shows track IDs in the trace list and "Trace" links on board cards

To enable tracing, add to your `config.json`:

```json
{
  "tracing_enabled": true
}
```

This sends traces via OTLP HTTP to `localhost:4318` (Jaeger all-in-one). Start Jaeger with:

```bash
docker run -d --name jaeger \
  -p 16686:16686 -p 4318:4318 \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/all-in-one:latest
```

View traces at `http://localhost:16686` or in the dashboard at `/-/dashboard/traces/{traceId}`.

The trace API is available at:
- `GET /-/api/traces` — list trace summaries (filter with `?track_id=X` or `?session_id=Y`)
- `GET /-/api/traces/{traceId}` — get full trace with span tree

## Origin Bridging

When you register a project with `kf add <remote-url>`, the remote URL is stored as the origin. This enables a future workflow: develop locally against Gitea (PRs, reviews, CI), then bridge changes back to your real remote (GitHub, GitLab) with a single command.

## Project Structure

```
kiloforge/
├── backend/          # Go backend (module: kiloforge)
│   ├── cmd/kf/       # CLI entrypoint
│   ├── internal/     # Clean architecture (adapter/, core/)
│   ├── go.mod
│   └── go.sum
├── frontend/         # React/Vite/TypeScript dashboard
│   ├── src/
│   ├── vite.config.ts
│   └── package.json
├── go.work           # Go workspace (IDE support)
└── Makefile          # Build orchestration
```

## Development

```bash
# Build everything (frontend + backend) into a single binary
make build

# Development mode: backend + Vite dev server with hot reload
make dev

# Run Go tests
make test

# Lint both Go and TypeScript
make lint

# Clean build artifacts
make clean
```

The `make dev` target starts the Go backend on port 3001 and the Vite dev server on port 5173. The Vite dev server proxies API calls to the backend, so you can develop the frontend with hot reload while hitting real backend endpoints.

## License

MIT
