# Architecture Overview

Kiloforge is a local orchestration platform with three main components: a **CLI**, the **Cortex** (control plane), and the **Command Deck** (dashboard). They work together to coordinate AI coding agents across multiple projects.

## System Diagram

```
                          ┌──────────────────────┐
                          │   Developer Machine   │
                          └──────────┬───────────┘
                                     │
              ┌──────────────────────┼──────────────────┐
              │                      │                  │
     ┌────────▼────────┐   ┌────────▼────────┐   ┌─────▼───────────┐
     │   kf CLI        │   │  Cortex         │   │  Claude Code    │
     │                 │   │  :4001          │   │  Agents         │
     │  up, down, add  │   │                 │   │                 │
     │  implement      │──►│  REST API       │   │  developer-1    │
     │  agents, logs   │   │  SSE events     │   │  developer-2    │
     │  stop, attach   │   │  WebSocket      │   │  reviewer-1     │
     └─────────────────┘   │  Agent spawner  │   │  interactive    │
                           │  Quota tracker  │   └─────────────────┘
                           │  Lock service   │
                           │                 │
                           │  ┌───────────┐  │
                           │  │ Cmd Deck  │  │
                           │  │ /-/       │  │
                           │  │ (React)   │  │
                           │  └───────────┘  │
                           └─────────────────┘
```

## Components

### CLI (`kf`)

The command-line interface is the primary entry point. It manages the lifecycle of the entire system:

- **`kf up`** — Start the Cortex control plane (first run performs setup automatically)
- **`kf down`** — Stop the Cortex
- **`kf add`** — Registers a project for agent orchestration
- **`kf implement`** — Spawns a developer agent for a specific track
- **`kf agents` / `kf logs` / `kf stop` / `kf attach`** — Agent management

The CLI is built with [Cobra](https://github.com/spf13/cobra) and compiles to a single Go binary with the Command Deck embedded.

### Cortex

The Cortex is the control plane — an HTTP server (port 4001) that serves multiple roles:

- **REST API** — OpenAPI 3.1 schema-first design with generated server interfaces. Handles agent CRUD, project management, quota queries, trace viewing, and board state.
- **SSE Event Bus** — Server-sent events for real-time updates. The Command Deck subscribes to agent status changes, quota updates, and project events.
- **WebSocket Terminal** — Interactive agent sessions. The Command Deck connects via WebSocket to send prompts and receive structured agent output (text, tool use, thinking, system messages).
- **Agent Spawner** — Manages agent lifecycle: spawn Claude Code processes, track PIDs, handle graceful shutdown, auto-recover on restart.
- **Quota Tracker** — In-memory token and cost tracking with JSON persistence. Receives usage data from the Claude Agent SDK and exposes it via API and SSE.
- **Scoped Lock Service** — HTTP-based distributed locks with TTL and heartbeat. Used by Conductor agents to serialize merges across worktrees.

### Command Deck

A React 19 / TypeScript / Vite application served at `/-/` by the Cortex:

- **Agent monitoring** — Real-time agent cards with status, tokens, cost, uptime
- **Interactive terminal** — WebSocket-based terminal for interactive agents with structured message display (text bubbles, tool use, thinking blocks)
- **Project management** — Add/remove projects, view sync status, push/pull from origin
- **Track board** — Kanban-style board showing track status (pending, in-progress, review, done)
- **Quota display** — Live token counts and estimated cost across all agents
- **Trace viewer** — OpenTelemetry trace timeline for track lifecycle visibility

The Command Deck builds to static files embedded in the Go binary via `go:embed`.

## Communication Protocols

| Path | Protocol | Purpose |
|------|----------|---------|
| CLI → Cortex | HTTP REST | Agent management, project operations |
| Command Deck → Cortex | HTTP REST + SSE | Data queries + real-time updates |
| Command Deck → Cortex | WebSocket | Interactive agent terminal |
| Cortex → Agents | Subprocess (stdin/stdout) | Agent spawning and SDK communication |

## Codebase Structure

```
kiloforge/
├── backend/                    # Go backend (module: kiloforge)
│   ├── cmd/kf/                 # CLI entrypoint and command registration
│   ├── internal/
│   │   ├── core/               # Domain layer (no external dependencies)
│   │   │   ├── domain/         # Domain types: AgentInfo, Project, Track, etc.
│   │   │   ├── port/           # Port interfaces: AgentStore, ProjectRegistry, etc.
│   │   │   └── service/        # Business logic: BoardService, etc.
│   │   └── adapter/            # Infrastructure layer
│   │       ├── agent/          # Agent spawner, quota tracker, SDK client
│   │       ├── cli/            # Cobra command implementations
│   │       ├── config/         # Configuration management
│   │       ├── dashboard/      # Command Deck server, SSE hub, watcher
│   │       ├── git/            # Git operations (worktrees, branches, sync)
│   │       ├── lock/           # Scoped lock service
│   │       ├── persistence/    # SQLite stores
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

A **track** is the unit of work. Each track has:
- A **spec** — what to build (acceptance criteria, context)
- A **plan** — how to build it (phased tasks)
- **Metadata** — status, type, timestamps

Tracks are managed by the Conductor workflow system. An architect creates tracks, developer agents claim and implement them, reviewer agents review PRs, and the system handles merging.

### Agents

An **agent** is a Claude Code CLI process managed by Kiloforge. Agents have:
- A **role** — developer, reviewer, or interactive
- A **status** — running, waiting, completed, failed, stopped
- A **session** — the Claude Code session ID, resumable after stop
- **Quota** — token counts and estimated cost

### Projects

A **project** is a registered git repository. Each project has:
- A local path on your machine
- Sync status tracking (ahead/behind origin)
- Worktree management for parallel agent work
- An origin remote URL for pushing/pulling

### Merge Lock

The merge lock ensures only one agent merges to `main` at a time. It supports two modes:
- **HTTP mode** — via the Cortex lock API with TTL (120s) and heartbeat (30s)
- **mkdir mode** — filesystem fallback when the Cortex is unreachable

## Data Storage

- **SQLite** — Primary storage for agents, projects, quota, traces, board state, tours, consent
- **JSON files** — Quota tracker persistence (`quota-usage.json`), configuration
- **Embedded assets** — Command Deck build artifacts compiled into the Go binary

## Tracing

OpenTelemetry distributed tracing follows each track through its lifecycle:

1. `kf implement` creates a root span `track/{trackId}`
2. Child spans track worktree acquisition, agent spawning, and session activity
3. The Command Deck displays traces with a timeline visualization

Traces are exported via OTLP HTTP to a local Jaeger instance (optional).
