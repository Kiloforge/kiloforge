# Getting Started

This guide walks the Kiloforger through installing Kiloforge, starting the Cortex, registering a project, spawning agents, and using the Command Deck.

## Prerequisites

- **Git** — `git` command available in PATH
- **Claude Code CLI** — `claude` command available in PATH, authenticated with an Anthropic API key or Claude Pro/Max subscription

## Install

### Homebrew (macOS/Linux)

```bash
brew tap Goblinlordx/tap
brew install kf
```

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Goblinlordx/crelay/main/install.sh | sh
```

### Build from Source

Requires Go 1.25+ and Node.js 18+.

```bash
git clone https://github.com/Goblinlordx/crelay.git
cd crelay
make build
# Binary at .build/kf — add to your PATH
```

## Start the Cortex

Run `kf up` to start the Cortex control plane:

```bash
kf up
```

On first run, this will:
1. Create the data directory (`~/.kiloforge/`)
2. Ask about anonymous usage analytics
3. Save configuration to `~/.kiloforge/config.json`
4. Start the Cortex daemon
5. Open the Command Deck in your browser

On subsequent runs, `kf up` simply starts the Cortex if it isn't already running.

This starts:
- **Cortex** at `http://localhost:4001` — the control plane (API, agent management, quota tracking, notifications)
- **Command Deck** at `http://localhost:4001/-/` — real-time monitoring dashboard

Stop everything with `kf down`.

## Register a Project

Add a project by providing its local path:

```bash
kf add /path/to/my-project
```

The Kiloforger can override the project slug:

```bash
kf add /path/to/my-project --name myapp
```

Or create a new project from scratch:

```bash
kf create myproject
```

List registered projects:

```bash
kf projects
```

## Set Up the Kiloforge Track System

Before spawning agents, the project needs Kiloforge artifacts. Open an interactive session and run the setup skill:

```bash
# From the Command Deck: spawn an interactive agent for your project
# Or from the CLI in your project directory:
claude -p "/kf-setup"
```

This initializes the project with:
- `product.md` — product definition and design principles
- `tech-stack.md` — technology choices and conventions
- `workflow.md` — TDD policy, commit strategy, verification commands
- `tracks.yaml` — track registry
- Code style guides

## Create Tracks with the Architect

Tracks are the unit of work in Kiloforge. Use the architect skill to create them:

```bash
# In your project directory:
claude -p "/kf-architect Add user authentication with OAuth2"
```

The architect will:
1. Research the codebase
2. Design the implementation approach
3. Create one or more tracks with specs and phased plans
4. Merge the track artifacts to main

Each track gets a unique ID like `feature/auth-oauth2_20260115100000Z`.

## Spawn Your First Agent

### From the CLI

Spawn a developer agent for a track:

```bash
kf implement <track-id>
```

This acquires a worktree from the pool, creates an implementation branch, and spawns a Claude Code agent running the `kf-developer` skill. The agent implements the track autonomously — writing code, running tests, and merging when complete.

Use `--list` to see available tracks:

```bash
kf implement --list
```

### From the Command Deck

1. Open `http://localhost:4001/-/` in your browser
2. Navigate to the project
3. View the track board — a kanban board showing tracks by status
4. Click on a track to spawn an agent directly from the UI

## The Command Deck

The Command Deck at `http://localhost:4001/-/` is the Kiloforger's control center for the Swarm.

### Agent Cards

Each agent appears as a card showing:
- **Status** — running, waiting, suspended, completed, failed
- **Role** — developer, reviewer, architect, advisor
- **Metrics** — token count, estimated cost, uptime
- **Actions** — stop, attach, view logs

### Interactive Terminals

The Command Deck supports interactive agent sessions. Open a terminal in the browser to:
- Send prompts to an interactive agent
- See structured output in real time (text, tool use, thinking blocks)
- Get notifications when an agent needs attention

### Track Board

A kanban-style board showing tracks across columns:
- **Pending** — tracks ready to be implemented
- **In Progress** — tracks being worked on by agents
- **Review** — tracks under review
- **Done** — completed tracks

Click any card to see track details, spawn agents, or view traces.

### Trace Viewer

When tracing is enabled, the Command Deck shows an OpenTelemetry trace timeline for each track. Click "Trace" on any board card to see the full lifecycle — from track creation through implementation, verification, and merge.

### Notification Center

The Command Deck displays notifications when agents need the Kiloforger's attention:
- **Waiting for input** — an interactive or worker agent has finished its turn and is waiting
- **Review needed** — a developer agent's PR needs review
- **Escalated** — a track has exceeded the review cycle limit

Notifications can be acknowledged or dismissed. They are delivered in real time via SSE.

### Swarm Panel

Monitor the Swarm's capacity and utilization:
- **Active agents** — how many agents are currently running
- **Available slots** — remaining capacity (configurable via `max_swarm_size`)
- **Worktree pool** — idle and in-use worktrees

## Monitor Your Swarm

### CLI

```bash
kf agents              # List all agents with status, role, cost
kf logs <agent-id>     # View agent logs (-f to follow)
kf stop <agent-id>     # Stop an agent (session preserved for resume)
kf attach <agent-id>   # Resume an agent's Claude session interactively
kf cost                # Show token usage and cost breakdown
kf pool                # Show worktree pool status
kf escalated           # Show tracks needing human intervention
kf status              # Show Cortex status and quota overview
```

### Shutdown

When the Kiloforger is done for the day:

```bash
kf down
```

This gracefully suspends all running agents and stops the Cortex. On the next `kf up`, suspended agents can be resumed with full session continuity.

## What's Next

- Read the [Architecture Overview](architecture.md) to understand how the pieces fit together
- Read [Skills Guide](skills.md) to learn about the full skills catalog and the architect → developer → reviewer pipeline
- Read [Agents & Swarms](agents-and-swarms.md) for deep coverage of agent lifecycle, suspension, notifications, and Swarm coordination
- Read [Why I Built This](why-i-built-this.md) for the story behind Kiloforge
- Explore the [OpenAPI Schema](../backend/api/openapi.yaml) for the full REST API
- Check the main [README](../README.md) for complete command reference
