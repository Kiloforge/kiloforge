# Getting Started

This guide walks through setting up Conductor Relay for a project from scratch.

## Prerequisites

### Required

1. **Docker** — Gitea runs as a container
   ```bash
   docker --version  # Must be installed and running
   ```

2. **Claude Code CLI** — The `claude` command must be in your PATH
   ```bash
   claude --version
   ```

3. **Go 1.21+** — To build crelay
   ```bash
   go version
   ```

4. **Git** — Your project must be a git repository

### Optional

- **`tea` CLI** — Gitea's official CLI for manual PR operations
  ```bash
  brew install tea  # macOS
  ```

## Installation

```bash
# Clone and build
cd ~/dev/crelay
make build

# Optionally install to PATH
cp crelay /usr/local/bin/
# or
go install ./cmd/crelay/
```

## Setup

### 1. Prepare Your Project

Your project should have Conductor artifacts already set up. If not:

```bash
cd ~/dev/my-project
claude -p "/conductor-setup"
```

This creates the `.agent/conductor/` directory with product definition, tech stack, workflow, and tracks.

### 2. Initialize Conductor Relay

From your project directory:

```bash
cd ~/dev/my-project
crelay init
```

You'll see output like:

```
==> Starting Gitea...
    Gitea running at http://localhost:3000
==> Configuring Gitea...
    Admin user: conductor
    Repository: conductor/my-project
==> Configuring git remote...
    Remote 'gitea' added
==> Registering webhooks...
    Webhook → http://host.docker.internal:3001/webhook
==> Starting relay server...
    Listening on http://localhost:3001

Ready. Gitea webhooks will spawn Claude agents automatically.
Press Ctrl+C to stop the relay (Gitea will keep running).
```

### 3. Verify

Open a new terminal:

```bash
# Check status
crelay status

# Visit Gitea web UI
open http://localhost:3000
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

## Usage Workflow

### Automatic: Webhook-Driven

1. **Generate tracks** (in your project):
   ```bash
   claude -p "/conductor-track-generator add user authentication"
   ```

2. **Start a developer agent** (in a worktree):
   ```bash
   claude --worktree developer-1 -p "/conductor-developer auth_20250307 --with-review --auto-merge"
   ```

3. **Watch it work** — The developer implements the track, creates a PR. The relay automatically spawns a reviewer. Check progress:
   ```bash
   crelay agents
   crelay logs <agent-id>
   ```

4. **Intervene if needed** — If an agent gets stuck or needs input:
   ```bash
   crelay attach <agent-id>
   # Then run the printed resume command in your terminal
   ```

### Manual: Using Gitea Directly

You can also interact with Gitea manually:

```bash
# Browse the web UI
open http://localhost:3000

# Use tea CLI (if installed)
tea login add --name local --url http://localhost:3000 --token <your-token>
tea pr list
tea pr view 1
```

## Multiple Developer Agents

To run multiple developers in parallel:

```bash
# Terminal 1: Developer 1
claude --worktree developer-1 -p "/conductor-developer track-1 --with-review"

# Terminal 2: Developer 2
claude --worktree developer-2 -p "/conductor-developer track-2 --with-review"

# Terminal 3: Monitor
crelay agents
```

Each developer works in an isolated worktree and creates separate PRs. The relay spawns a reviewer for each PR.

## Teardown

```bash
# Stop relay: Ctrl+C in the init terminal

# Remove everything
crelay destroy        # stop container, remove git remote
crelay destroy --data # also delete Gitea data and logs
```

## Troubleshooting

### Gitea won't start

```bash
# Check Docker
docker ps -a | grep conductor-gitea
docker logs conductor-gitea

# Retry
docker rm -f conductor-gitea
crelay init
```

### Webhook not firing

```bash
# Check relay is running
curl http://localhost:3001/health

# Check webhook config in Gitea UI
open http://localhost:3000/conductor/my-project/settings/hooks

# Check relay logs (in the init terminal)
```

### Agent won't start

```bash
# Verify claude is available
which claude
claude --version

# Check agent logs
crelay logs <agent-id>

# Check state file
cat ~/.crelay/state.json
```

### Port conflicts

```bash
# Use different ports
crelay init --gitea-port 3010 --relay-port 3011
```
