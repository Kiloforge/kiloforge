# Getting Started

This guide walks through setting up Kiloforge.

## Prerequisites

### Required

1. **Docker with Docker Compose** — Gitea runs as a compose service
   ```bash
   docker compose version  # v2 (Docker Desktop)
   # OR
   docker-compose version  # v1 (Colima/standalone)
   ```

2. **Claude Code CLI** — The `claude` command must be in your PATH
   ```bash
   claude --version
   ```

3. **Go 1.21+** — To build kiloforge
   ```bash
   go version
   ```

### Colima Users

If you're using Colima on macOS, install docker-compose separately:

```bash
brew install docker-compose
```

Kiloforge auto-detects which compose variant is available (v2 first, v1 fallback).

### Optional

- **`tea` CLI** — Gitea's official CLI for manual PR operations
  ```bash
  brew install tea  # macOS
  ```

## Installation

```bash
# Clone and build
cd ~/dev/kiloforge
make build

# Optionally install to PATH
cp kf /usr/local/bin/
# or
go install ./cmd/kf/
```

## Setup

### 1. Initialize the Global Gitea Server

```bash
kf init
```

You'll see output like:

```
==> Detecting Docker Compose...
    Found: 2.29.1
==> Generating docker-compose.yml...
==> Starting Gitea...
    Gitea running at http://localhost:4000
==> Configuring Gitea...
    Admin user: conductor

Gitea is ready!
  Web UI:     http://localhost:4000
  Admin:      conductor / conductor123
  Data:       /Users/you/.kiloforge
  Compose:    /Users/you/.kiloforge/docker-compose.yml

Stop with 'kf down', restart with 'kf up'.
Register a project with 'kf add <path>'.
```

### 2. Register a Project

```bash
cd ~/dev/my-project
kf add .
```

This creates a Gitea repo, adds a `gitea` remote, pushes your main branch, and registers a webhook. The origin remote URL is captured for future bridging.

### 3. Verify

```bash
# Check status
kf status

# Visit Gitea web UI
open http://localhost:4000
# Login: conductor / conductor123
```

### 4. Set Environment Variables

For conductor skills to use Gitea automatically, set these in your shell or `.env`:

```bash
export CONDUCTOR_REMOTE=gitea
export CONDUCTOR_PR_PLATFORM=gitea
```

Or add to your project's `.claude/settings.json`:
```json
{
  "env": {
    "CONDUCTOR_REMOTE": "gitea",
    "CONDUCTOR_PR_PLATFORM": "gitea"
  }
}
```

## Usage

### Conductor Developer Workflow

1. **Generate tracks** (in your project):
   ```bash
   claude -p "/conductor-track-generator add user authentication"
   ```

2. **Start a developer agent** (in a worktree):
   ```bash
   claude --worktree developer-1 -p "/conductor-developer auth_20250307 --with-review --auto-merge"
   ```

3. **Watch it work** — The developer implements the track and creates a PR.

### Manual Gitea Interaction

```bash
# Browse the web UI
open http://localhost:4000

# Use tea CLI (if installed)
tea login add --name local --url http://localhost:4000 --token <your-token>
tea pr list
```

## Daily Use

```bash
kf down    # stop Gitea (keeps data)
kf up      # start Gitea again
```

## Teardown

```bash
kf destroy          # permanently delete everything (requires confirmation)
kf destroy --force  # skip confirmation prompt
```

## Troubleshooting

### Gitea won't start

```bash
# Check Docker/compose
docker compose -f ~/.kiloforge/docker-compose.yml ps
docker compose -f ~/.kiloforge/docker-compose.yml logs

# Retry
kf destroy --force
kf init
```

### Port conflicts

```bash
# Use a different port
kf init --gitea-port 3010
```

### Colima: host.docker.internal not resolving

The generated compose file includes `extra_hosts: ["host.docker.internal:host-gateway"]` which should handle this. If you still have issues, check that your Docker Engine is version 20.10+.
