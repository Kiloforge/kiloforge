# Architecture Overview

Kiloforge is a local orchestration platform with three main components: a **CLI**, the **Cortex** (control plane), and the **Command Deck** (dashboard). They work together to coordinate AI coding agents — the Swarm — across multiple projects.

## System Diagram

```
                          ┌──────────────────────┐
                          │   Kiloforger's Machine │
                          └──────────┬───────────┘
                                     │
              ┌──────────────────────┼──────────────────┐
              │                      │                  │
     ┌────────▼────────┐   ┌────────▼────────┐   ┌─────▼───────────┐
     │   kf CLI        │   │  Cortex         │   │  Claude Code    │
     │                 │   │  :39517         │   │  Swarm          │
     │  up, down, add  │   │                 │   │                 │
     │  implement      │──►│  REST API       │   │  developer-1    │
     │  agents, logs   │   │  SSE events     │   │  developer-2    │
     │  stop, attach   │   │  WebSocket      │   │  reviewer-1     │
     │  cost, skills   │   │  Agent spawner  │   │  architect      │
     └─────────────────┘   │  Quota tracker  │   │  interactive    │
                           │  Lock service   │   └─────────────────┘
                           │  Notification   │
                           │  Skills mgr     │
                           │                 │
                           │  ┌───────────┐  │
                           │  │ Cmd Deck  │  │
                           │  │ /         │  │
                           │  │ (React)   │  │
                           │  └───────────┘  │
                           └─────────────────┘
```

## Components

### CLI (`kf`)

The command-line interface is the Kiloforger's primary entry point. It manages the lifecycle of the entire system:

- **`kf up`** — Start the Cortex control plane (first run performs setup automatically)
- **`kf down`** — Stop the Cortex (agents are gracefully suspended)
- **`kf add` / `kf create`** — Register or create projects for agent orchestration
- **`kf implement`** — Spawn a developer agent for a specific track
- **`kf agents` / `kf logs` / `kf stop` / `kf attach`** — Agent management
- **`kf cost`** — Token usage and cost breakdown
- **`kf skills`** — Manage and update skills
- **`kf push`** — Push changes to the origin remote
- **`kf pool`** — Worktree pool status
- **`kf escalated`** — Tracks needing the Kiloforger's intervention

The CLI is built with [Cobra](https://github.com/spf13/cobra) and compiles to a single Go binary with the Command Deck embedded.

### Cortex

The Cortex is the control plane — an HTTP server (port 39517) that serves multiple roles:

- **REST API** — OpenAPI 3.1 schema-first design with generated server interfaces. Handles agent CRUD, project management, quota queries, trace viewing, board state, and swarm capacity.
- **SSE Event Bus** — Server-sent events for real-time updates. The Command Deck subscribes to agent status changes, quota updates, notifications, and capacity events.
- **WebSocket Terminal** — Interactive agent sessions. The Command Deck connects via WebSocket to send prompts and receive structured agent output (text, tool use, thinking, system messages).
- **Agent Spawner** — Manages agent lifecycle: spawn Claude Code processes, track PIDs, handle graceful shutdown, auto-recover on restart. Enforces Swarm capacity limits.
- **Quota Tracker** — In-memory token and cost tracking with JSON persistence. Receives usage data from the Claude Agent SDK and exposes it via API and SSE.
- **Scoped Lock Service** — HTTP-based distributed locks with TTL (120s) and heartbeat (30s). Used by agents to serialize merges across worktrees.
- **Notification Service** — Creates and manages agent-needs-attention notifications. Deduplicates alerts, supports acknowledge and dismiss operations, and delivers events via SSE.
- **Skills Manager** — Validates and installs agent skills. Checks that required skills are available before spawning agents.

### Command Deck

A React 19 / TypeScript / Vite application served at `/` by the Cortex:

- **Agent monitoring** — Real-time agent cards with status, role, tokens, cost, uptime
- **Interactive terminal** — WebSocket-based terminal for interactive agents with structured message display (text bubbles, tool use, thinking blocks)
- **Project management** — Add/remove projects, view sync status, push/pull from origin
- **Track board** — Kanban-style board showing track status (pending, in-progress, review, done)
- **Quota display** — Live token counts and estimated cost across all agents
- **Trace viewer** — OpenTelemetry trace timeline for track lifecycle visibility
- **Notification center** — Real-time alerts when agents need the Kiloforger's attention
- **Swarm panel** — Capacity monitoring, worktree pool status, and queue management
- **Agent launcher** — Spawn agents directly from the UI
- **Admin operations** — Reliability dashboard, settings management

The Command Deck builds to static files embedded in the Go binary via `go:embed`.

## Communication Protocols

| Path | Protocol | Purpose |
|------|----------|---------|
| CLI → Cortex | HTTP REST | Agent management, project operations |
| Command Deck → Cortex | HTTP REST + SSE | Data queries + real-time updates |
| Command Deck → Cortex | WebSocket | Interactive agent terminal |
| Cortex → Agents | Subprocess (stdin/stdout) | Agent spawning and SDK communication |

## Skills System

Skills are structured slash commands that agents use for specific workflows. They are installed as markdown files in `~/.claude/skills/` and invoked by agents during their work.

### Skill Categories

| Category | Skills | Purpose |
|----------|--------|---------|
| Core Workflow | kf-architect, kf-developer, kf-implement, kf-reviewer, kf-dispatch | The main development pipeline |
| Management | kf-manage, kf-bulk-archive, kf-compact-archive, kf-revert | Track lifecycle operations |
| Review & Advisory | kf-advisor-product, kf-advisor-reliability | Strategic analysis and recommendations |
| Setup & Onboarding | kf-setup, kf-getting-started, kf-interactive, kf-new-track | Project initialization |
| Infrastructure | kf-validate, kf-repair, kf-report, kf-data-guardian | System health and integrity |

### The Pipeline

The core development workflow follows a structured pipeline:

1. **Architect** (`/kf-architect`) — Researches the codebase, designs implementation, creates tracks with specs and plans
2. **Developer** (`/kf-developer`) — Claims a track, creates a branch, implements all tasks following the plan, merges to main
3. **Reviewer** (`/kf-reviewer`) — Reviews a developer's PR against the track spec and project standards (optional, via `--with-review`)

Skills are validated before agent spawn — the Cortex checks that all required skills are installed and up to date.

See [Skills Guide](skills.md) for the full catalog with descriptions.

## Notification Bus

The notification system alerts the Kiloforger when agents need attention:

### Triggers

- **Interactive agent waiting** — When an interactive or advisor agent finishes its turn and waits for input, a notification is created
- **Worker agent waiting** — The dashboard watcher scans worker agent (developer/reviewer) logs every 2 seconds; if it detects a `turn_end` event, it creates a notification
- **Auto-dismiss** — When an agent starts a new turn (`turn_start`), the notification is automatically dismissed
- **Cleanup** — When an agent reaches a terminal status (completed, failed, stopped), all its notifications are cleaned up

### Delivery

Notifications are delivered to the Command Deck via SSE events (`notification_created`, `notification_dismissed`). The Kiloforger can acknowledge or dismiss them from the notification center.

### Deduplication

Only one active notification exists per agent at any time. If an agent already has an unacknowledged notification, no duplicate is created.

## Swarm Coordination

The Swarm is the collective of Claude Code agents managed by the Cortex.

### Capacity

Swarm capacity is configurable via `max_swarm_size` (default: 3). The Cortex enforces this limit — `kf implement` and interactive agent spawn requests are rejected when the Swarm is at capacity. Capacity changes are published as SSE events.

### Worktree Pool

Each developer agent runs in an isolated git worktree. The pool manager:
- Maintains a set of worktree slots for the project
- Acquires an idle worktree when `kf implement` is called
- Resets the worktree to main and creates an implementation branch
- Returns the worktree to the pool when the agent completes
- Auto-commits and stashes incomplete work for interrupted agents

Pool status is visible via `kf pool` and in the Command Deck's Swarm panel.

### Queue Service

The queue service handles dependency-aware track scheduling:
- Reads `deps.yaml` and only enqueues tracks whose dependencies are satisfied
- Uses topological sorting to order ready tracks
- Enforces concurrency limits via semaphore
- Monitors agent lifecycle: enqueue → spawn → monitor → complete → return worktree

### Agent Suspension

The Cortex manages agent suspension to conserve resources:

- **Worker roles** (developer, reviewer) — never auto-suspend. They run autonomously without a browser connection.
- **Interactive roles** (architect, advisor, interactive) — auto-suspend after a configurable grace period (default: 30 seconds) when the Kiloforger disconnects from the WebSocket session.
- **Resume** — Suspended agents can be resumed with `kf attach` or via the Command Deck. Full session continuity is preserved.
- **Shutdown** — `kf down` gracefully suspends all running agents. On restart, they can be resumed.

See [Agents & Swarms](agents-and-swarms.md) for full lifecycle documentation.

## Codebase Structure

```
kiloforge/
├── backend/                    # Go backend (module: kiloforge)
│   ├── cmd/kf/                 # CLI entrypoint and command registration
│   ├── internal/
│   │   ├── core/               # Domain layer (no external dependencies)
│   │   │   ├── domain/         # Domain types: AgentInfo, Project, Track, etc.
│   │   │   ├── port/           # Port interfaces: AgentStore, ProjectRegistry, etc.
│   │   │   └── service/        # Business logic: BoardService, QueueService, NotificationService
│   │   └── adapter/            # Infrastructure layer
│   │       ├── agent/          # Agent spawner, quota tracker, SDK client, suspension
│   │       ├── cli/            # Cobra command implementations
│   │       ├── config/         # Configuration management
│   │       ├── dashboard/      # Command Deck server, SSE hub, watcher
│   │       ├── git/            # Git operations (worktrees, branches, sync)
│   │       ├── lock/           # Scoped lock service
│   │       ├── persistence/    # SQLite stores
│   │       ├── pool/           # Worktree pool management
│   │       ├── rest/           # REST API handler, OpenAPI gen
│   │       ├── skills/         # Embedded skill management
│   │       ├── tracing/        # OpenTelemetry integration
│   │       └── ws/             # WebSocket hub for interactive sessions
│   ├── api/
│   │   ├── openapi.yaml        # OpenAPI 3.1 schema (source of truth)
│   │   └── asyncapi.yaml       # AsyncAPI 3.0 event documentation
│   └── go.mod
├── frontend/                   # React/Vite/TypeScript Command Deck
│   ├── src/
│   │   ├── api/                # Fetcher, query keys, error handling
│   │   ├── components/         # Reusable UI components
│   │   ├── hooks/              # Custom React hooks (useAgents, useWebSocket, etc.)
│   │   ├── pages/              # Route pages
│   │   └── types/              # TypeScript type definitions
│   └── vite.config.ts
├── go.work                     # Go workspace (IDE module resolution)
├── Makefile                    # Build orchestration
└── docs/                       # Project documentation
```

## Key Abstractions

### Tracks

A **track** is the unit of work in the Kiloforge track system. Each track has:
- A **spec** — what to build (acceptance criteria, context, codebase analysis)
- A **plan** — how to build it (phased tasks)
- **Metadata** — status, type, timestamps, dependencies

Tracks are created by the architect skill, implemented by developer agents, and optionally reviewed by reviewer agents. The system handles dependency ordering, conflict detection, and merge serialization.

### Agents

An **agent** is a Claude Code CLI process managed by the Cortex. Agents have:
- A **role** — developer, reviewer, architect, advisor-product, advisor-reliability, or interactive
- A **status** — running, waiting, halted, suspended, completed, failed, stopped, force-killed, resume-failed, or replaced
- A **session** — the Claude Code session ID, resumable after stop or suspension
- **Quota** — token counts and estimated cost

### Projects

A **project** is a registered git repository. Each project has:
- A local path on the Kiloforger's machine
- Sync status tracking (ahead/behind origin)
- Worktree management for parallel agent work
- An origin remote URL for pushing/pulling

### Merge Lock

The merge lock ensures only one agent merges to main at a time across the entire Swarm. It supports two modes:
- **HTTP mode** — via the Cortex lock API with TTL (120s) and heartbeat (30s). Preferred when the Cortex is running.
- **mkdir mode** — filesystem fallback when the Cortex is unreachable

## Data Storage

- **SQLite** — Primary storage for agents, projects, quota, traces, board state, notifications, consent
- **JSON files** — Quota tracker persistence (`quota-usage.json`), worktree pool state (`pool.json`), configuration
- **Embedded assets** — Command Deck build artifacts compiled into the Go binary

## Tracing

OpenTelemetry distributed tracing follows each track through its lifecycle:

1. `kf implement` creates a root span `track/{trackId}`
2. Child spans track worktree acquisition, agent spawning, and session activity
3. Track events (implementation, verification, merge) join the trace via stored trace IDs
4. The Command Deck displays traces with a timeline visualization

Traces are exported via OTLP HTTP to a local Jaeger instance (optional). See the [main README](../README.md#tracing) for setup instructions.
