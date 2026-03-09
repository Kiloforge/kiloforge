# Getting Started

This guide walks you through installing Kiloforge, setting up the infrastructure, registering your first project, and spawning your first agent.

## Prerequisites

- **Git** — `git` command available in PATH
- **Docker** with Docker Compose — Docker Desktop (includes compose v2) or Docker Engine + `docker-compose` (v1 for Colima users)
- **Claude Code CLI** — `claude` command available in PATH, authenticated with an Anthropic API key or Claude Pro/Max subscription

### Colima Users (macOS)

If using Colima instead of Docker Desktop:

```bash
brew install docker-compose
```

Both `docker compose` (v2) and `docker-compose` (v1) are auto-detected.

## Install

### Homebrew (macOS/Linux)

```bash
brew tap Goblinlordx/tap
brew install kf
```

### Build from Source

Requires Go 1.25+ and Node.js 18+.

```bash
git clone https://github.com/Goblinlordx/crelay.git
cd crelay
make build
# Binary at .build/kf — add to your PATH
```

## Initialize

Run `kf init` to start the local infrastructure:

```bash
kf init
```

This will:
1. Generate a `docker-compose.yml` in `~/.kiloforge/`
2. Start a Gitea instance at `http://localhost:4000`
3. Create an admin user with a random password
4. Register your SSH key for git operations
5. Save configuration to `~/.kiloforge/config.json`

The admin password is displayed once and saved to config. Subsequent runs are idempotent.

## Start the Orchestrator

For daily use, start Gitea and the orchestrator together:

```bash
kf up
```

This starts:
- **Gitea** at `http://localhost:4000` — the local git forge
- **Orchestrator** at `http://localhost:4001` — API, dashboard, and agent management
- **Dashboard** at `http://localhost:4001/-/` — real-time monitoring UI

Stop everything with `kf down`.

## Register a Project

Add a project by providing its remote URL:

```bash
kf add git@github.com:user/my-project.git
```

This clones the repo locally, creates a mirror on Gitea, pushes all branches, and registers a webhook for event routing. You can also use HTTPS URLs:

```bash
kf add https://github.com/user/my-project.git
```

List registered projects:

```bash
kf projects
```

## Spawn Your First Agent

### From the CLI

If your project uses the Conductor track system, spawn a developer agent for a track:

```bash
kf implement <track-id>
```

### From the Dashboard

1. Open `http://localhost:4001/-/` in your browser
2. Navigate to your project
3. Use the track board to view available tracks
4. Spawn agents directly from the UI

### Interactive Sessions

The dashboard supports interactive agent sessions — open a terminal in the browser, send prompts, and see structured output (text, tool use, thinking blocks) in real time.

## Monitor

### Dashboard

The dashboard at `http://localhost:4001/-/` shows:
- **Agent cards** — status, role, tokens, cost, uptime for each agent
- **Quota** — total token usage and estimated cost across all agents
- **Logs** — live log streaming for any agent
- **Traces** — OpenTelemetry trace timeline for track lifecycle

### CLI

```bash
kf agents          # List all agents
kf logs <agent-id> # View agent logs (-f to follow)
kf stop <agent-id> # Stop an agent (session preserved)
kf attach <agent-id> # Resume an agent's Claude session
```

## What's Next

- Read the [Architecture Overview](architecture.md) to understand how the pieces fit together
- Read the [Author's Foreword](foreword.md) for the story behind Kiloforge
- Explore the [OpenAPI schema](../backend/api/openapi.yaml) for the full REST API
- Check the main [README](../README.md) for complete command reference
