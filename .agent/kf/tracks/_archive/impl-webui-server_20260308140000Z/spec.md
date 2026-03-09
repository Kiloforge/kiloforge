# Specification: Web UI Server with Real-Time Agent Monitoring

**Track ID:** impl-webui-server_20260308140000Z
**Type:** Feature
**Created:** 2026-03-08T14:00:00Z
**Status:** Draft

## Summary

Add a standalone web dashboard server that provides real-time visibility into agent status, quota usage, track progress, and log streaming. Entirely additive — does not modify any existing relay, spawner, or CLI behavior.

## Context

Kiloforge currently operates as a pure CLI tool with a webhook relay server. All monitoring requires CLI commands (`kf status`, `kf agents`, `kf logs`). A web dashboard would provide real-time visibility without polling CLI commands, especially valuable when multiple agents are running concurrently.

The dashboard is an **additive, read-only observer** — it reads existing state files (state.json, quota-usage.json, pr-tracking.json) and exposes them via a web interface with live updates via SSE.

## Codebase Analysis

- **Relay server**: `internal/relay/server.go` — HTTP on port 3001 with `/webhook` and `/health`. Not modified.
- **State store**: `internal/state/state.go` / `internal/adapter/persistence/jsonfile/agent_store.go` — agent status, session IDs, PIDs
- **Quota tracker**: `internal/agent/tracker.go` — `GetAgentUsage()`, `GetTotalUsage()`, `IsRateLimited()` (completed track)
- **PR tracking**: `internal/adapter/persistence/jsonfile/pr_tracking_store.go` — PR lifecycle state
- **Config**: `internal/config/` — port/adapter resolution chain
- **No existing UI code** — no HTML, JS, CSS, or templates anywhere in the project

### Additive design

- New package: `internal/dashboard/` (or `internal/adapter/dashboard/`)
- Separate HTTP server on its own port (default: 3002)
- Reads state via existing Go APIs — no file locking conflicts
- SSE endpoint for push updates — no polling overhead
- Embedded static assets via `embed.FS` — single binary, no external files

## Acceptance Criteria

- [ ] `internal/dashboard/server.go` — standalone HTTP server on configurable port
- [ ] `internal/dashboard/handlers.go` — REST API endpoints for agent/quota/track data
- [ ] `internal/dashboard/sse.go` — SSE endpoint for real-time state change notifications
- [ ] `internal/dashboard/static/` — embedded HTML/CSS/JS dashboard
- [ ] Dashboard displays: agent list with status, role, track, PID, uptime
- [ ] Dashboard displays: quota usage (per-agent and aggregate tokens, cost)
- [ ] Dashboard displays: rate limit status with visual indicator
- [ ] Dashboard displays: track progress (from tracks.md or track metadata)
- [ ] SSE pushes updates when agent status changes, quota updates, or errors occur
- [ ] Log viewer: stream agent log files in browser (tail -f style)
- [ ] Links to Gitea: PRs, repos, user profile (configurable Gitea URL)
- [ ] `go:embed` for static assets — no external file dependencies
- [ ] Unit tests for API handlers (JSON responses, edge cases)
- [ ] Dashboard is entirely optional — relay works without it
- [ ] All existing tests pass, build succeeds

## Dependencies

- impl-quota-tracker_20260307160000Z (completed — provides quota data APIs)

## Blockers

- **impl-webui-integration_20260308140001Z** — depends on this track for the dashboard server

## Conflict Risk

- **LOW** — Entirely new package (`internal/dashboard/`). No modifications to existing files. Reads state via existing APIs.
- Pending tracks `refactor-clean-arch` and `test-coverage-alignment` don't conflict since this is a new package.
- `impl-graceful-shutdown-recovery` is independent — dashboard shutdown is handled separately.

## Out of Scope

- Modifying the relay server or webhook handling
- Modifying agent spawner behavior
- Authentication/authorization for the dashboard
- Reverse proxy to Gitea (just link to it)
- Write operations from the dashboard (stop/start agents) — future track
- Mobile-responsive design (desktop-first)

## Technical Notes

### Server architecture

```go
// internal/dashboard/server.go

type Server struct {
    port     int
    store    *state.Store      // read agent state
    tracker  *agent.QuotaTracker // read quota data
    giteaURL string            // for linking
    hub      *SSEHub           // broadcast state changes
}

func New(port int, store *state.Store, tracker *agent.QuotaTracker, giteaURL string) *Server
func (s *Server) Run(ctx context.Context) error
```

### API endpoints

```
GET /api/agents         — list all agents with status
GET /api/agents/:id     — single agent details
GET /api/agents/:id/log — tail agent log file
GET /api/quota          — aggregate and per-agent quota usage
GET /api/tracks         — track progress summary
GET /api/status         — overall system status
GET /events             — SSE stream for real-time updates
GET /                   — serve embedded SPA
```

### SSE hub design

```go
type SSEHub struct {
    mu      sync.RWMutex
    clients map[chan SSEEvent]struct{}
}

type SSEEvent struct {
    Type string      `json:"type"` // "agent_update", "quota_update", "error"
    Data interface{} `json:"data"`
}

func (h *SSEHub) Broadcast(event SSEEvent)
func (h *SSEHub) Subscribe() <-chan SSEEvent
func (h *SSEHub) Unsubscribe(ch <-chan SSEEvent)
```

### Frontend approach

Minimal vanilla JS + CSS embedded via `go:embed`. No build step, no npm, no framework. The dashboard is a single HTML page that:
- Fetches initial state via REST endpoints
- Subscribes to `/events` SSE for live updates
- Updates DOM on events
- Uses CSS grid for layout

This keeps the Go-only toolchain and avoids frontend build complexity.

### State polling

The dashboard server polls state files periodically (every 2-3 seconds) and broadcasts changes via SSE. This is simpler than modifying existing code to push events:

```go
func (s *Server) watchState(ctx context.Context) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.checkAndBroadcastChanges()
        }
    }
}
```

---

_Generated by conductor-track-generator_
