# Specification: Interactive Agent Sessions via WebSocket (Backend)

**Track ID:** interactive-agent-be_20260309150000Z
**Type:** Feature
**Created:** 2026-03-09T15:00:00Z
**Status:** Draft

## Summary

Add WebSocket-based bidirectional communication with Claude Code agent processes. Interactive agents connect stdin/stdout to a WebSocket endpoint, allowing dashboard users to have real-time conversations with agents for setup, track creation, and manual intervention.

## Context

Currently all agents are non-interactive — they spawn with a `-p` prompt, run to completion, and log output to a file. There's no way to send input to a running agent from the dashboard. However, several workflows benefit from interactive conversation:

1. **Setup wizard** — `kf init` could be guided interactively
2. **Track creation** — users want back-and-forth with a track generator agent
3. **Manual intervention** — attach to a paused agent from the dashboard instead of running `kf attach` in a terminal

Claude Code CLI reads from stdin when running interactively. By piping stdin/stdout through a WebSocket, the dashboard becomes a terminal for the agent.

## Codebase Analysis

### Current spawner (`agent/spawner.go`)
- `cmd.StdoutPipe()` already used for output capture
- `cmd.Stderr = logFile` for error logging
- `cmd.Stdin` is NOT set (defaults to /dev/null)
- `monitorAgent()` reads stdout line-by-line, parses stream-json, logs to file

### WebSocket support
- No WebSocket currently — only SSE (one-way) via `dashboard/sse.go`
- Go stdlib `net/http` supports WebSocket upgrade via `golang.org/x/net/websocket` or `nhooyr.io/websocket`

### Session management
- Each agent has a `SessionID` (UUID) for resume capability
- `--resume <session-id>` continues a previous conversation
- Can combine with `-p` to inject a new prompt on resume

### Output format
- `--output-format stream-json` produces machine-parseable JSON lines
- For interactive use, we likely want **raw text output** (not stream-json) so the user sees natural conversation
- Or: parse stream-json and extract the text content to display

## Acceptance Criteria

- [ ] `SpawnInteractive()` method creates agent with stdin pipe connected
- [ ] WebSocket endpoint `GET /ws/agent/{id}` upgrades to WebSocket
- [ ] User messages sent via WebSocket are written to agent's stdin
- [ ] Agent stdout is streamed back through the WebSocket
- [ ] Non-interactive agents (developer/reviewer) are unaffected — existing spawn methods unchanged
- [ ] Agent log file still captures all output (interactive and non-interactive)
- [ ] WebSocket disconnection does NOT kill the agent — agent continues running, can reconnect
- [ ] Multiple WebSocket clients can observe the same agent (read-only for additional clients)
- [ ] `go test ./...` passes
- [ ] `make build` succeeds

## Dependencies

None.

## Blockers

None.

## Conflict Risk

- **sqlite-storage-core_20260309140000Z** — LOW. Different subsystems.
- **origin-sync-api_20260309143000Z** — LOW. Different subsystems.

## Out of Scope

- Frontend terminal UI (separate FE track)
- Interactive `kf init` wizard (separate track, builds on this infrastructure)
- Interactive track generator (separate track, builds on this infrastructure)
- PTY emulation (raw stdin/stdout is sufficient for Claude Code)

## Technical Notes

### WebSocket library
Use `nhooyr.io/websocket` — lightweight, idiomatic Go, supports context cancellation.

### Interactive spawn
```go
func (s *Spawner) SpawnInteractive(ctx context.Context, opts InteractiveOpts) (*InteractiveAgent, error) {
    cmd := exec.CommandContext(ctx, "claude", args...)
    cmd.Dir = opts.WorkDir

    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    cmd.Stderr = logFile

    cmd.Start()

    return &InteractiveAgent{
        ID:     agentID,
        Stdin:  stdin,
        Stdout: stdout,
        Cmd:    cmd,
    }, nil
}
```

### WebSocket endpoint
```
GET /ws/agent/{id}
  → Upgrade to WebSocket
  → Binary/text frames relayed to/from agent stdin/stdout
  → Close frame on agent exit
```

### Message protocol
Simple text-based protocol over WebSocket:
```json
// Client → Server (user input)
{"type": "input", "text": "Yes, use the default settings"}

// Server → Client (agent output)
{"type": "output", "text": "Setting up Gitea container..."}

// Server → Client (agent status)
{"type": "status", "status": "running"}
{"type": "status", "status": "completed", "exit_code": 0}

// Server → Client (error)
{"type": "error", "message": "Agent process exited unexpectedly"}
```

### Reconnection
When a WebSocket client disconnects:
- Agent continues running (stdin stays open)
- Output is buffered (ring buffer, last N lines) for reconnecting clients
- New WebSocket connection replays buffered output then streams live

### Output format for interactive agents
Use `--output-format stream-json` still, but parse and extract human-readable content. The stream-json format includes `assistant` message events with text content. Parse these and forward the text to the WebSocket. This gives us both:
- Structured data for quota tracking (same as non-interactive)
- Human-readable text for the UI

### Concurrency model
- One goroutine reads WebSocket → writes to stdin
- One goroutine reads stdout → writes to WebSocket + log file
- Agent process lifecycle managed by existing spawner infrastructure
- WebSocket handler registered on the main HTTP mux

---

_Generated by conductor-track-generator from prompt: "interactive agent sessions for setup, track creation, and manual intervention"_
