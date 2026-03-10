# Architecture Overview

Kiloforge is a local orchestration platform with four main components: a **CLI**, an **orchestrator server**, a **web dashboard**, and a **Gitea instance**. They work together to coordinate AI coding agents across multiple projects.

## System Diagram

```
                          ┌──────────────────────┐
                          │   Developer Machine   │
                          └──────────┬───────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
     ┌────────▼────────┐   ┌────────▼────────┐   ┌────────▼────────┐
     │   kf CLI        │   │  Orchestrator   │   │  Gitea (Docker) │
     │                 │   │  :4001          │   │  :4000          │
     │  init, up, add  │   │                 │◄──│                 │
     │  implement      │──►│  REST API       │   │  Git repos      │
     │  agents, logs   │   │  SSE events     │   │  Pull requests  │
     │  stop, attach   │   │  WebSocket      │   │  Code review    │
     └─────────────────┘   │  Agent spawner  │   │  Webhooks ──────┤
                           │  Quota tracker  │   │                 │
                           │  Lock service   │   └─────────────────┘
                           │                 │
                           │  ┌───────────┐  │   ┌─────────────────┐
                           │  │ Dashboard │  │   │  Claude Code    │
                           │  │ /-/       │  │   │  Agents         │
                           │  │ (React)   │  │   │                 │
                           │  └───────────┘  │   │  developer-1    │
                           └─────────────────┘   │  developer-2    │
                                                 │  reviewer-1     │
                                                 │  interactive    │
                                                 └─────────────────┘
```

## Components

### CLI (`kf`)

The command-line interface is the primary entry point. It manages the lifecycle of the entire system:

- **`kf init`** — First-time setup: starts Gitea via Docker Compose, creates admin user, generates API token
- **`kf up` / `kf down`** — Daily start/stop of Gitea and the orchestrator
- **`kf add`** — Registers a project: clones repo, creates Gitea mirror, sets up webhook
- **`kf implement`** — Spawns a developer agent for a specific track
- **`kf agents` / `kf logs` / `kf stop` / `kf attach`** — Agent management

The CLI is built with [Cobra](https://github.com/spf13/cobra) and compiles to a single Go binary with the React dashboard embedded.

### Orchestrator Server

The orchestrator is an HTTP server (port 4001) that serves multiple roles:

- **REST API** — OpenAPI 3.1 schema-first design with generated server interfaces. Handles agent CRUD, project management, quota queries, trace viewing, and board state.
- **SSE Event Bus** — Server-sent events for real-time updates. The dashboard subscribes to agent status changes, quota updates, and project events.
- **WebSocket Terminal** — Interactive agent sessions. The dashboard connects via WebSocket to send prompts and receive structured agent output (text, tool use, thinking, system messages).
- **Webhook Receiver** — Processes Gitea webhooks for PR events, reviews, and pushes. Routes events by repository to the correct project.
- **Agent Spawner** — Manages agent lifecycle: spawn Claude Code processes, track PIDs, handle graceful shutdown, auto-recover on restart.
- **Quota Tracker** — In-memory token and cost tracking with JSON persistence. Receives usage data from the Claude Agent SDK and exposes it via API and SSE.
- **Scoped Lock Service** — HTTP-based distributed locks with TTL and heartbeat. Used by Conductor agents to serialize merges across worktrees.
- **Gitea Reverse Proxy** — Proxies requests to the Gitea container with automatic authentication injection.

### Web Dashboard

A React 19 / TypeScript / Vite application served at `/-/` by the orchestrator:

- **Agent monitoring** — Real-time agent cards with status, tokens, cost, uptime
- **Interactive terminal** — WebSocket-based terminal for interactive agents with structured message display (text bubbles, tool use, thinking blocks)
- **Project management** — Add/remove projects, view sync status, push/pull from origin
- **Track board** — Kanban-style board showing track status (pending, in-progress, review, done)
- **Quota display** — Live token counts and estimated cost across all agents
- **Trace viewer** — OpenTelemetry trace timeline for track lifecycle visibility

The dashboard builds to static files embedded in the Go binary via `go:embed`.

### Gitea

A self-hosted git forge running in Docker:

- Provides git repositories, pull requests, and code review
- Fires webhooks to the orchestrator on PR, review, and push events
- Agents push branches, open PRs, and merge — all on localhost
- Reverse-proxy authentication eliminates password management

## Communication Protocols

| Path | Protocol | Purpose |
|------|----------|---------|
| CLI → Orchestrator | HTTP REST | Agent management, project operations |
| Dashboard → Orchestrator | HTTP REST + SSE | Data queries + real-time updates |
| Dashboard → Orchestrator | WebSocket | Interactive agent terminal |
| Gitea → Orchestrator | HTTP Webhooks | PR, review, push event notifications |
| Orchestrator → Gitea | HTTP API | Repo creation, PR management |
| Orchestrator → Agents | Subprocess (stdin/stdout) | Agent spawning and SDK communication |

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
│   │       ├── dashboard/      # Dashboard server, SSE hub, watcher
│   │       ├── git/            # Git operations (worktrees, branches)
│   │       ├── gitea/          # Gitea API client
│   │       ├── lock/           # Scoped lock service
│   │       ├── persistence/    # SQLite stores
│   │       ├── rest/           # REST API handler, OpenAPI gen
│   │       ├── tracing/        # OpenTelemetry integration
│   │       └── ws/             # WebSocket hub for interactive sessions
│   ├── api/
│   │   ├── openapi.yaml        # OpenAPI 3.1 schema (source of truth)
│   │   └── asyncapi.yaml       # AsyncAPI 3.0 event documentation
│   └── go.mod
├── frontend/                   # React/Vite/TypeScript dashboard
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
- A local clone in `~/.kiloforge/repos/`
- A mirror on the Gitea instance
- Webhook routing for event handling
- An origin remote URL for bridging back to GitHub/GitLab

### Merge Lock

The merge lock ensures only one agent merges to `main` at a time. It supports two modes:
- **HTTP mode** — via the orchestrator's lock API with TTL (120s) and heartbeat (30s)
- **mkdir mode** — filesystem fallback when the orchestrator is unreachable

## Data Storage

- **SQLite** — Primary storage for agents, projects, quota, traces, board state, tours, consent
- **JSON files** — Quota tracker persistence (`quota-usage.json`), legacy state
- **Docker volumes** — Gitea data (repos, database)
- **Embedded assets** — Frontend build artifacts compiled into the Go binary

## Tracing

OpenTelemetry distributed tracing follows each track through its lifecycle:

1. `kf implement` creates a root span `track/{trackId}`
2. Child spans track worktree acquisition, agent spawning, and session activity
3. Webhook events (PR opened, review submitted, merge) join the trace via stored trace IDs
4. The dashboard displays traces with a timeline visualization

Traces are exported via OTLP HTTP to a local Jaeger instance (optional).
