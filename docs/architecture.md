# Architecture

## Overview

Conductor Relay is a bridge between Gitea (git forge) and Claude Code (AI agent). It receives webhook events from Gitea and translates them into agent lifecycle actions — spawning, monitoring, halting, and resuming Claude Code sessions.

## Components

### 1. CLI (`internal/cli/`)

Cobra-based command-line interface. Each command is a separate file:

- **`init.go`** — Orchestrates the full startup sequence: Docker, Gitea config, git remote, webhooks, relay server
- **`status.go`** — Queries Docker and relay health, displays summary
- **`agents.go`** — Lists tracked agents from state file
- **`logs.go`** — Reads agent log files, supports follow mode
- **`attach.go`** — Halts a running agent (SIGINT) and prints the `claude --resume` command
- **`stop.go`** — Halts an agent without providing resume instructions
- **`destroy.go`** — Tears down Docker container and optionally removes data

### 2. Gitea Manager (`internal/gitea/manager.go`)

Handles Docker container lifecycle:

- **Start**: Checks if container exists (running/stopped/absent), starts or creates as needed
- **WaitReady**: Polls the Gitea API until it responds (up to 60s)
- **Configure**: Creates admin user via `docker exec`, creates API token and repository via REST API
- **SetupGitRemote**: Adds `gitea` remote to the project, pushes `main`

### 3. Gitea Client (`internal/gitea/client.go`)

Thin wrapper around Gitea's REST API. Handles:

- Authentication (token or basic auth)
- Token creation
- Repository creation
- Webhook registration
- PR fetching

All API calls go through a single `do()` method that handles JSON serialization, auth headers, and error responses.

### 4. Relay Server (`internal/relay/server.go`)

HTTP server that receives Gitea webhooks and dispatches agent actions:

**Webhook Handlers:**

| Event | Action | Handler |
|-------|--------|---------|
| `pull_request` (opened) | Spawn reviewer agent | `handlePullRequest` |
| `pull_request` (synchronize) | Log update, optionally re-trigger review | `handlePullRequest` |
| `pull_request_review` | Log review state, notify developer (future) | `handlePullRequestReview` |
| `pull_request_comment` | Log comment | `handlePullRequestComment` |

**API Endpoints:**

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/webhook` | POST | Receive Gitea webhooks |
| `/health` | GET | Health check |
| `/api/agents` | GET | List agents (JSON) |

### 5. Agent Spawner (`internal/agent/spawner.go`)

Manages Claude Code process lifecycle:

- **SpawnReviewer**: Launches `claude -p "/conductor-reviewer <pr-url>"` with a new session ID
- **SpawnDeveloper**: Launches `claude -p "/conductor-developer <track> <flags>"` with a new session ID

Each spawned agent:
1. Gets a UUID-based agent ID and session ID
2. Runs with `--output-format stream-json` for structured output
3. Has stdout captured to a log file in `~/.crelay/logs/`
4. Is tracked in the state file with PID, session ID, status

### 6. State Store (`internal/state/state.go`)

JSON file-based state persistence. Tracks:

- Agent ID, role, reference (track/PR), status
- Claude session ID (for resume)
- Process PID (for signaling)
- Worktree directory
- Timestamps

Operations: Load, Save, AddAgent, FindAgent (prefix match), UpdateStatus, HaltAgent (SIGINT).

### 7. Config (`internal/config/config.go`)

JSON configuration stored at `~/.crelay/config.json`. Contains ports, paths, repo name, and API token.

## Data Flow

```
Developer runs:
  /conductor-developer track-123 --with-review
    │
    ├─ Implements track
    ├─ git push gitea feature/track-123
    ├─ tea pr create (or API call)
    │
    ▼
Gitea receives push + PR creation
    │
    ├─ Fires pull_request.opened webhook
    │
    ▼
Relay receives POST /webhook
    │
    ├─ Parses event type and PR details
    ├─ Calls spawner.SpawnReviewer(prNumber, prURL)
    │     │
    │     ├─ Generates agent ID + session ID
    │     ├─ Creates log file
    │     ├─ Starts: claude -p "/conductor-reviewer <url>" --session-id <uuid> --output-format stream-json
    │     ├─ Records PID in state
    │     └─ Streams stdout to log file (goroutine)
    │
    ▼
Reviewer agent runs
    │
    ├─ Fetches PR diff via Gitea API
    ├─ Reviews against track spec
    ├─ Posts review (approve/request changes) via API
    │
    ▼
Gitea fires pull_request_review webhook
    │
    ├─ Relay logs the review event
    ├─ (Future: automatically notify developer agent)
    │
    ▼
User checks status:
  crelay agents    → sees developer waiting, reviewer completed
  crelay attach <developer-id>  → halts developer, gets resume command
  claude --resume <session-id>  → takes over developer session interactively
```

## Process Isolation

Each Claude agent runs as a separate OS process. The relay tracks PIDs for signaling:

- **SIGINT** — Graceful halt. Claude saves session state and exits. Session can be resumed.
- **Process exit** — Relay detects via `cmd.Wait()` in the output goroutine. Updates status to `completed` or `failed`.

## Future Considerations

### PTY-Based Agents

The current stream-json approach is one-directional (relay reads output, cannot send input). A PTY-based approach would allow:

- Writing to agent stdin (sending review feedback, merge commands)
- True "attach" (connecting terminal to running agent)
- Richer log capture (including prompts and formatting)

This would use `github.com/creack/pty` to allocate pseudo-terminals.

### TUI Dashboard

A terminal UI built with [`tview`](https://github.com/rivo/tview) (the library behind k9s) for:
- Real-time agent log streaming
- Agent lifecycle controls (stop, attach)
- Gitea PR status overview
- Split-pane view: agent list + selected agent logs

### Multi-Project Support

Currently one relay per project. Could be extended to manage multiple projects with a project selector.
