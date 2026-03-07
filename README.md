# crelay

A local collaboration platform for AI agents and humans. Provides a fast, private workspace with a git forge (Gitea), a real-time monitoring dashboard, and orchestration tools for managing diverse agents at scale — all running efficiently on your machine.

## Why

Working with multiple AI agents across multiple projects demands infrastructure that is fast, observable, and under your control. Remote forges add latency, rate limits, and cost. crelay gives you:

- **Fast, local-first** — zero network latency for git operations, webhooks, and agent coordination
- **Human + AI collaboration** — Gitea for code review and PRs, plus a custom web dashboard for real-time agent monitoring, quota tracking, and log streaming
- **Agent orchestration at scale** — spawn, monitor, throttle, suspend, and resume dozens of concurrent agents across multiple projects
- **Session persistence** — gracefully shut down agents and auto-recover them on restart, with full session continuity
- **Quota-aware** — track token usage and cost per agent/track, enforce budgets, and handle rate limits gracefully
- **Extensible** — scoped lock service, webhook relay, and REST APIs that agents and tools can build on
- **Full control** — everything runs on your machine, no external dependencies

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
crelay add git@github.com:user/my-project.git

# List registered projects
crelay projects
```

This will:
1. Detect your Docker Compose CLI variant (v2 or v1)
2. Generate a `docker-compose.yml` in `~/.crelay/`
3. Start a Gitea instance at `http://localhost:3000`
4. Create an admin user (`conductor` / random password)
5. Generate an API token and save config
6. Register your project: create Gitea repo, add remote, push code

## Commands

### `crelay init`

One-time setup: start the global Gitea server via Docker Compose.

```bash
crelay init [flags]

Flags:
  --gitea-port int    Port for Gitea web UI (default 3000)
  --data-dir string   Persistent data directory (default ~/.crelay)
  --admin-pass string Admin password (default: generated random)
  --ssh-key string    Path to SSH public key (default: auto-detect)
```

On first init, a random admin password is generated and saved to `config.json`. Subsequent runs reuse the saved password. Use `--admin-pass` to override.

Your SSH public key is auto-detected from `~/.ssh/` (tries `id_ed25519.pub`, `id_rsa.pub`, `id_ecdsa.pub`) and registered with the Gitea admin user. Use `--ssh-key` to specify a custom path. Missing SSH keys produce a warning but do not prevent initialization.

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

Clone a remote repo and register it with the Gitea server.

```bash
crelay add git@github.com:user/repo.git          # SSH URL
crelay add https://github.com/user/repo.git      # HTTPS URL
crelay add git@github.com:user/repo.git --name x  # override slug
```

Clones the remote into `~/.crelay/repos/<slug>/`, creates a Gitea repo, adds a `gitea` remote, pushes the main branch, and registers a webhook.

### `crelay projects`

List registered projects.

```bash
crelay projects
```

### `crelay implement`

Approve a conductor track and spawn a developer agent in a pooled worktree.

```bash
crelay implement <track-id>            # spawn developer for track
crelay implement --list                # list available tracks
crelay implement --project myapp <id>  # specify project explicitly
```

The command acquires a worktree from the pool, prepares it (reset to main, create implementation branch), and spawns a Claude Code agent running `/conductor-developer <track-id>`. Agent state is recorded for monitoring with `crelay agents`, `crelay logs`, `crelay stop`, and `crelay attach`.

### `crelay agents`

List active and recent agents.

```bash
crelay agents          # table output
crelay agents --json   # JSON output
```

### `crelay logs <agent-id>`

View logs for an agent. Supports prefix matching on the agent ID.

```bash
crelay logs abc12345
crelay logs abc12345 -f   # follow mode
```

### `crelay stop <agent-id>`

Send SIGINT to stop a running agent. The session is preserved for later resume.

### `crelay attach <agent-id>`

Print the command to resume an agent's Claude session interactively. If the agent is running, it is halted first.

### `crelay pool`

Show worktree pool status. Displays idle and in-use worktrees for developer agents.

```bash
crelay pool
```

### `crelay escalated`

Show PRs that hit the review cycle limit and require human intervention.

```bash
crelay escalated
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
    ├─ Webhook relay server (localhost:3001)
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
    └─ crelay add: register project → Gitea repo + webhook

┌─────────────────────────────────────────────────────────────┐
│  Gitea (Docker)                            localhost:3000    │
│  • Git repos, PRs, code review for multiple projects        │
│  • Webhooks → relay on events                               │
└────────────────────────┬────────────────────────────────────┘
                         │ webhooks
┌────────────────────────▼────────────────────────────────────┐
│  Relay Server                              localhost:3001    │
│  • Multi-project event routing                              │
│  • Developer-reviewer relay cycle                           │
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

All persistent data lives in `~/.crelay/` (configurable via `--data-dir`):

```
~/.crelay/
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

## Origin Bridging

When you register a project with `crelay add <remote-url>`, the remote URL is stored as the origin. This enables a future workflow: develop locally against Gitea (PRs, reviews, CI), then bridge changes back to your real remote (GitHub, GitLab) with a single command.

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
