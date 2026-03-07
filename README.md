# Conductor Relay

A local Gitea instance + webhook relay server that orchestrates Claude Code agents for [Conductor](https://github.com/your-org/ai-skills) workflows.

One command starts everything: a Gitea server, a webhook relay, and automatic agent spawning for code review and development.

## Why

Conductor's role-based agents (developer, reviewer) work best with a git forge for PRs and code review. Running Gitea locally gives you:

- **Free, private, fast** — no GitHub rate limits or network latency
- **Automatic agent orchestration** — webhooks trigger Claude agents (reviewer spawns when a PR is opened)
- **Session management** — view logs, halt agents, and resume their Claude sessions interactively
- **Full control** — everything runs on your machine

## Prerequisites

- **Docker** — for running Gitea
- **Claude Code CLI** — `claude` command available in PATH
- **Go 1.21+** — to build (or use prebuilt binary)

## Quick Start

```bash
# Build
make build

# Navigate to your project (must be a git repo with conductor set up)
cd ~/dev/my-project

# Initialize everything
conductor-relay init

# That's it. Gitea is running, webhooks are registered, relay is listening.
```

This will:
1. Start a Gitea instance at `http://localhost:3000`
2. Create an admin user (`conductor` / `conductor123`)
3. Create a repo mirroring your project
4. Add a `gitea` git remote
5. Push your code to Gitea
6. Register webhooks for PR events
7. Start the relay server on port 3001

## Commands

### `conductor-relay init`

Start Gitea and the relay server. Run this from your project directory.

```bash
conductor-relay init [flags]

Flags:
  --gitea-port int   Port for Gitea web UI (default 3000)
  --relay-port int   Port for the relay server (default 3001)
  --repo string      Repository name (default: directory name)
  --data-dir string  Persistent data directory (default ~/.conductor-relay)
```

### `conductor-relay status`

Show the status of Gitea, the relay server, and active agents.

```bash
$ conductor-relay status
Conductor Relay Status
======================
Gitea:       running (http://localhost:3000)
Relay:       running (http://localhost:3001)
Project:     /Users/you/dev/my-project
Repository:  conductor/my-project
Data:        /Users/you/.conductor-relay
Agents:      2 active
```

### `conductor-relay agents`

List all tracked agents (active and recent).

```bash
$ conductor-relay agents
ID        ROLE       TRACK/PR    STATUS    SESSION   STARTED
a1b2c3d4  developer  auth_track  running   e5f6g7h8  14:23:01
i9j0k1l2  reviewer   PR #3       waiting   m3n4o5p6  14:25:12

$ conductor-relay agents --json  # JSON output
```

### `conductor-relay logs <agent-id>`

View an agent's log output. Supports prefix matching on the ID.

```bash
$ conductor-relay logs a1b2c3d4
$ conductor-relay logs a1b2 -f  # follow mode
```

### `conductor-relay attach <agent-id>`

Halt a running agent and get the command to resume its Claude session interactively. Use this when you need to provide manual input or guidance.

```bash
$ conductor-relay attach a1b2c3d4
Agent:     a1b2c3d4 (developer)
Status:    running
Session:   e5f6g7h8-...

This agent is currently running. It will be sent SIGINT to pause it.
After it stops, resume with:

  cd /Users/you/dev/my-project && claude --resume e5f6g7h8-...

Agent halted. You can now resume it with the command above.
```

### `conductor-relay stop <agent-id>`

Gracefully stop an agent without attaching.

```bash
$ conductor-relay stop a1b2c3d4
Agent a1b2c3d4 stopped.
Resume with: claude --resume e5f6g7h8-...
```

### `conductor-relay destroy`

Tear down Gitea and clean up.

```bash
conductor-relay destroy          # stop container, remove remote
conductor-relay destroy --data   # also delete persistent data
```

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│  Your Project (git repo with conductor artifacts)        │
│                                                          │
│  git remote: gitea → http://localhost:3000/conductor/repo│
└───────────────────────┬──────────────────────────────────┘
                        │ git push / PR create
                        ▼
┌──────────────────────────────────────────────────────────┐
│  Gitea (Docker)                          localhost:3000   │
│                                                          │
│  • Hosts git repos                                       │
│  • Manages PRs and reviews                               │
│  • Sends webhooks on events                              │
└───────────────────────┬──────────────────────────────────┘
                        │ POST /webhook
                        ▼
┌──────────────────────────────────────────────────────────┐
│  Relay Server                            localhost:3001   │
│                                                          │
│  Webhook handlers:                                       │
│  • pull_request.opened   → spawn reviewer agent          │
│  • pull_request_review   → notify developer agent        │
│  • pull_request.synchronize → re-trigger reviewer        │
│                                                          │
│  Agent management:                                       │
│  • Tracks PIDs, session IDs, logs                        │
│  • GET  /health         → relay health check             │
│  • GET  /api/agents     → list agents (JSON)             │
│                                                          │
│  Each agent runs as:                                     │
│  claude -p "<skill>" --session-id <uuid>                 │
│         --output-format stream-json                      │
└───────────────────────┬──────────────────────────────────┘
                        │ spawns
                        ▼
┌──────────────────────────────────────────────────────────┐
│  Claude Code Agents                                      │
│                                                          │
│  • conductor-developer: implements tracks, creates PRs   │
│  • conductor-reviewer: reviews PRs, approves/requests    │
│                                                          │
│  Session preserved in ~/.claude/projects/                 │
│  Resumable with: claude --resume <session-id>            │
└──────────────────────────────────────────────────────────┘
```

## Workflow

### Automated PR Review

1. Developer agent runs `/conductor-developer <track> --with-review`
2. Developer implements the track and creates a PR on Gitea
3. Gitea fires `pull_request.opened` webhook
4. Relay spawns a reviewer agent: `claude -p "/conductor-reviewer <pr-url>"`
5. Reviewer posts review via Gitea API
6. Developer is notified (or manually resumed via `attach`)

### Manual Intervention

At any point you can:
- **Check status**: `conductor-relay agents` to see what's running
- **Read logs**: `conductor-relay logs <id>` to see what an agent is doing
- **Take over**: `conductor-relay attach <id>` to halt and resume interactively
- **Stop**: `conductor-relay stop <id>` to pause an agent

## Configuration

### Environment Variables

The conductor skills respect these env vars (set automatically by the relay):

| Variable | Value | Purpose |
|----------|-------|---------|
| `CONDUCTOR_REMOTE` | `gitea` | Git remote name for push/PR |
| `CONDUCTOR_PR_PLATFORM` | `gitea` | PR platform detection |

### Data Directory

All persistent data lives in `~/.conductor-relay/` (configurable via `--data-dir`):

```
~/.conductor-relay/
├── config.json       # Relay configuration
├── state.json        # Agent tracking state
├── logs/             # Agent log files (one per agent)
│   ├── <agent-id>.log
│   └── ...
└── gitea-data/       # Gitea Docker volume (repos, DB)
```

## Current Limitations

- **No automatic developer notification**: When a reviewer posts feedback, the developer agent must be manually resumed via `attach`. Future versions will support PTY-based agents with stdin injection.
- **Single project**: Each relay instance manages one project. Run multiple instances on different ports for multiple projects.
- **No TUI for logs**: Logs are file-based. A `tview`-based terminal dashboard is planned.
- **`tea` CLI not yet integrated**: The relay uses Gitea's REST API directly. The `tea` CLI can be installed separately for manual interactions.

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
