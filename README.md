# Kiloforge

**1,000x Productivity.** The Kiloforger's forge — command AI agent swarms and ship code at the speed of thought.

An orchestration platform for coordinating AI coding agents at scale. Runs the Cortex control plane, Command Deck, and Claude Code swarms directly on your machine — transforming pure intent into meaningful action.

## Why

Coordinating multiple AI agents across multiple projects demands infrastructure that is observable, automated, and under your control. Kiloforge gives the Kiloforger:

- **Private infrastructure, cloud AI** — the Cortex and all coordination run locally; agents are Claude Code CLI sessions powered by Anthropic's cloud APIs
- **Kiloforger + Swarm** — direct collaboration between Kiloforgers and Claude Code swarms via tracks, worktrees, and the Command Deck
- **Agent orchestration at scale** — spawn, monitor, throttle, suspend, and resume dozens of concurrent agents across multiple projects via the Cortex
- **Session persistence** — gracefully shut down agents and auto-recover them on restart, with full session continuity
- **Quota-aware** — track token usage and cost per agent/track, enforce budgets, and handle rate limits gracefully
- **End-to-end tracing** — OpenTelemetry traces follow each track from conception through agent work, verification, and merge
- **Extensible** — scoped lock service, webhook relay, and REST APIs that agents and tools can build on
- **Full control** — your code stays on your machine; only requires Git and Claude Code

## Installation

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Goblinlordx/crelay/main/install.sh | sh
```

To install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/Goblinlordx/crelay/main/install.sh | INSTALL_DIR=~/.local/bin sh
```

### Homebrew (macOS/Linux)

```bash
brew tap Goblinlordx/tap
brew install kf
```

### Binary Download

Download the latest release from [GitHub Releases](https://github.com/Goblinlordx/crelay/releases). Archives are available for:

| OS | Arch | Archive |
|----|------|---------|
| macOS | Intel | `kf_*_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `kf_*_darwin_arm64.tar.gz` |
| Linux | x86_64 | `kf_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `kf_*_linux_arm64.tar.gz` |
| Windows | x86_64 | `kf_*_windows_amd64.zip` |
| Windows | ARM64 | `kf_*_windows_arm64.zip` |

Extract and place `kf` in your `PATH`.

### Build from Source

```bash
git clone https://github.com/Goblinlordx/crelay.git
cd crelay
make build
# Binary at .build/kf
```

## Prerequisites

- **Git** — `git` command available in PATH
- **Claude Code CLI** — `claude` command available in PATH

### Building from Source

- **Go 1.25+**
- **Node.js 18+**

## Quick Start

```bash
# Start the Cortex (first run performs setup automatically)
kf up

# Register your project
kf add /path/to/my-project

# List registered projects
kf projects

# Spawn a developer agent for a track
kf implement <track-id>

# Monitor your Swarm
kf agents
```

This will:
1. Create the data directory (`~/.kiloforge/`)
2. Save the global configuration
3. Start the Cortex control plane on `localhost:4001`
4. Open the Command Deck in your browser
5. Register your project for agent orchestration

## Commands

### Cortex Lifecycle

#### `kf up`

Start the Cortex control plane. On first run, this performs one-time setup (creates data directory, saves configuration). Returns immediately once the Cortex is running.

```bash
kf up [--host string] [--port int]
```

#### `kf down`

Stop the Cortex. Active agents are gracefully suspended and can be resumed on the next `kf up`.

```bash
kf down
```

#### `kf status`

Show Cortex status, quota usage, and agent costs.

```bash
$ kf status
Kiloforge Status — Kiloforger
================================
Cortex:      running (PID 12345) on :4001
Data:        /Users/you/.kiloforge
Server:      http://localhost:4001
Dashboard:   http://localhost:4001/-/
```

#### `kf destroy`

Permanently destroy all Kiloforge data (requires confirmation).

```bash
kf destroy          # prompts for confirmation
kf destroy --force  # skip confirmation
```

### Project Management

#### `kf add`

Register a project with the Cortex. Clones the repository to a managed location and sets up worktree pooling.

```bash
kf add /path/to/project              # local path
kf add /path/to/project --name x     # override slug
```

#### `kf create`

Create a new project from scratch.

```bash
kf create myproject
```

#### `kf projects`

List registered projects.

```bash
kf projects
```

#### `kf push`

Push changes from the internal clone to the origin remote.

```bash
kf push [slug]              # push current branch
kf push slug --branch name  # push specific branch
kf push --all               # push all projects
```

### Agent Management

#### `kf implement`

Spawn a developer agent in a pooled worktree for a specific track.

```bash
kf implement <track-id>            # spawn developer for track
kf implement --list                # list available tracks
kf implement --project myapp <id>  # specify project explicitly
kf implement --dry-run <id>        # preview without spawning
```

The command acquires a worktree from the pool, prepares it (reset to main, create implementation branch), and spawns a Claude Code agent running `/kf-developer <track-id>`. Agent state is recorded for monitoring with `kf agents`, `kf logs`, `kf stop`, and `kf attach`.

#### `kf agents`

List active and recent agents.

```bash
kf agents          # table output
kf agents --json   # JSON output
```

#### `kf logs <agent-id>`

View logs for an agent. Supports prefix matching on the agent ID.

```bash
kf logs abc12345
kf logs abc12345 -f   # follow mode
```

#### `kf stop <agent-id>`

Send SIGINT to stop a running agent. The session is preserved for later resume.

#### `kf attach <agent-id>`

Print the command to resume an agent's Claude session interactively. If the agent is running, it is halted first.

#### `kf cost`

Show token usage and estimated cost per agent.

```bash
kf cost
```

#### `kf escalated`

Show tracks that hit the review cycle limit and require the Kiloforger's intervention.

```bash
kf escalated
```

### Swarm & Worktrees

#### `kf pool`

Show worktree pool status. Displays idle and in-use worktrees for the Swarm.

```bash
kf pool
```

### Skills

#### `kf skills`

Manage Kiloforge skills — the slash commands that agents use for structured workflows.

```bash
kf skills list     # list installed skills
kf skills update   # update to latest version
```

### Dashboard

#### `kf dashboard`

Start the Command Deck standalone (without the full Cortex).

```bash
kf dashboard
```

### Sync

#### `kf sync`

Sync Kiloforge tracks to the native board representation.

```bash
kf sync
```

## Architecture

```
kf up
    │
    ├─ Cortex (localhost:4001)
    │   ├─ Agent lifecycle: spawn, suspend, resume
    │   ├─ Quota tracking and budget enforcement
    │   ├─ Scoped lock API (merge serialization)
    │   ├─ Notification bus (agent-needs-attention alerts)
    │   ├─ Skills management and validation
    │   └─ Track and worktree coordination
    │
    ├─ Command Deck (localhost:4001/-/)
    │   ├─ Real-time agent status via SSE
    │   ├─ Interactive agent terminals (WebSocket)
    │   ├─ Kanban track board
    │   ├─ Trace viewer (OpenTelemetry)
    │   ├─ Quota/cost monitoring
    │   ├─ Notification center
    │   └─ Swarm capacity panel
    │
    └─ Claude Code Swarm
        ├─ Autonomous agents in pooled worktrees
        ├─ Structured skills (architect → developer → reviewer)
        └─ Implement, verify, and merge directly
```

## Data Directory

All persistent data lives in `~/.kiloforge/` (configurable via `--data-dir`):

```
~/.kiloforge/
├── config.json           # Global configuration
├── projects.json         # Project registry
├── pool.json             # Worktree pool state
├── state.json            # Agent state (running/completed agents)
├── projects/             # Per-project data
│   └── <slug>/
│       └── logs/             # Agent log files
└── orchestrator.log      # Cortex daemon log
```

## Tracing

Kiloforge supports OpenTelemetry distributed tracing with **track lifecycle tracing** — a single trace follows a development track from conception through agent work, verification, and merge. This gives end-to-end visibility into the full lifecycle of every track.

When enabled:
- **`kf implement`** creates a root span `track/{trackId}` with child spans for worktree acquisition, agent spawning, and session tracking
- **Track events** (implementation, verification, merge) automatically join the track's trace via stored trace IDs, so all activity for a track appears in one trace
- **Agent spans** include `session.id` attributes for cross-referencing with Claude Code sessions
- **The Command Deck** shows track IDs in the trace list and "Trace" links on board cards

To enable tracing, add to your `config.json`:

```json
{
  "tracing_enabled": true
}
```

This sends traces via OTLP HTTP to `localhost:4318` (Jaeger all-in-one). Start Jaeger with:

```bash
docker run -d --name jaeger \
  -p 16686:16686 -p 4318:4318 \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/all-in-one:latest
```

View traces at `http://localhost:16686` or in the Command Deck at `/-/dashboard/traces/{traceId}`.

The trace API is available at:
- `GET /-/api/traces` — list trace summaries (filter with `?track_id=X` or `?session_id=Y`)
- `GET /-/api/traces/{traceId}` — get full trace with span tree

## Analytics / Telemetry

Kiloforge collects anonymous usage data via [PostHog](https://posthog.com) to help improve the product. Telemetry is **opt-out** — enabled by default, but easy to disable.

### What is collected

- CLI command invocations (command name only)
- Server startup events (version, OS, architecture)
- Agent lifecycle events (spawned/completed, role, duration)
- Project registration events (clone vs create — no URLs or names)

**No PII is ever collected.** The device identifier is a one-way SHA-256 hash of hostname + data directory — not reversible.

### How to opt out

There are three ways to disable analytics:

1. **During first run** — Answer "n" to the analytics prompt:
   ```
   Help improve kiloforge by sending anonymous usage data? (Y/n) n
   ```

2. **Command Deck toggle** — Open the settings menu (gear icon) and toggle "Anonymous usage data" off.

3. **Environment variable** — Set before running any command:
   ```bash
   export KF_ANALYTICS_ENABLED=false
   ```

When disabled, a no-op tracker is used — no network requests are made.

## Project Structure

```
kiloforge/
├── backend/          # Go backend (module: kiloforge)
│   ├── cmd/kf/       # CLI entrypoint
│   ├── internal/     # Clean architecture (adapter/, core/)
│   ├── go.mod
│   └── go.sum
├── frontend/         # React/Vite/TypeScript Command Deck
│   ├── src/
│   ├── vite.config.ts
│   └── package.json
├── go.work           # Go workspace (IDE support)
└── Makefile          # Build automation
```

## Development

```bash
# Build everything (frontend + backend) into a single binary
make build

# Development mode: backend + Vite dev server with hot reload
make dev

# Run Go tests
make test

# Lint both Go and TypeScript
make lint

# Clean build artifacts
make clean
```

The `make dev` target starts the Go backend on port 3001 and the Vite dev server on port 5173. The Vite dev server proxies API calls to the backend, so the Kiloforger can develop the frontend with hot reload while hitting real backend endpoints.

## Releasing

Releases are automated via GoReleaser and GitHub Actions.

```bash
# Tag a release
git tag v0.1.0
git push origin v0.1.0
```

This triggers the release workflow which builds binaries for all platforms, creates a GitHub Release with checksums, and updates the Homebrew tap.

**Required GitHub secrets:**
- `HOMEBREW_TAP_TOKEN` — PAT with write access to the `Goblinlordx/homebrew-tap` repo

To test locally:

```bash
make release-local   # goreleaser --snapshot --clean
```

## Documentation

- [Why I Built This](docs/why-i-built-this.md) — The story behind Kiloforge
- [Getting Started](docs/getting-started.md) — Installation, first agent, and Command Deck walkthrough
- [Architecture Overview](docs/architecture.md) — How the pieces fit together
- [Skills Guide](docs/skills.md) — The full skills catalog and workflow pipeline
- [Agents & Swarms](docs/agents-and-swarms.md) — Agent lifecycle, notifications, and Swarm coordination

## Contributing

Kiloforge is not currently accepting external pull requests — they are automatically closed by a GitHub Actions workflow. All development is managed by the maintainer using AI agent orchestration.

To contribute, please [open an issue](https://github.com/Goblinlordx/crelay/issues/new/choose):

- **Bug reports** — describe the problem, steps to reproduce, and your environment
- **Feature requests** — describe the problem you're solving and your proposed solution

The maintainer will assess each issue and manage implementation through the internal development workflow.

## Developer Notes

This project started on 2026-03-07 at 17:03 KST and is still in its early phase. Even so, and even though there may be issues found, I believe that regardless, this serves as a blueprint to replicate the way I have been working. I built Kiloforge without Kiloforge — primarily through the use of the skills contained in this repository: `/kf-setup`, `/kf-architect`, and `/kf-developer`. This project is intended to specifically address all the challenges I ran into when attempting to manually orchestrate 6-8 AI agents at a time, and in a way that should be fun, easy, and effortless.

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

This project includes derivative works of [gemini-conductor](https://github.com/goblinlordx/gemini-conductor) (MIT). See [NOTICE](NOTICE) for attribution.
