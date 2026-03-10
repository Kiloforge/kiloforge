# Getting Started

This guide walks you through installing Kiloforge, starting the Cortex, registering your first project, and spawning your first agent.

## Prerequisites

- **Git** — `git` command available in PATH
- **Claude Code CLI** — `claude` command available in PATH, authenticated with an Anthropic API key or Claude Pro/Max subscription

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
- **Cortex** at `http://localhost:4001` — the control plane (API, agent management, quota tracking)
- **Command Deck** at `http://localhost:4001/-/` — real-time monitoring dashboard

Stop everything with `kf down`.

## Register a Project

Add a project by providing its local path:

```bash
kf add /path/to/my-project
```

You can override the project slug:

```bash
kf add /path/to/my-project --name myapp
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

### From the Command Deck

1. Open `http://localhost:4001/-/` in your browser
2. Navigate to your project
3. Use the track board to view available tracks
4. Spawn agents directly from the UI

### Interactive Sessions

The Command Deck supports interactive agent sessions — open a terminal in the browser, send prompts, and see structured output (text, tool use, thinking blocks) in real time.

## Monitor

### Command Deck

The Command Deck at `http://localhost:4001/-/` shows:
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
- Read [Why I Built This](why-i-built-this.md) for the story behind Kiloforge
- Explore the [OpenAPI schema](../backend/api/openapi.yaml) for the full REST API
- Check the main [README](../README.md) for complete command reference
