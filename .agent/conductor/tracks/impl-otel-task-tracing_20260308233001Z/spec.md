# Specification: OpenTelemetry Task-Level Tracing and Token Metrics

**Track ID:** impl-otel-task-tracing_20260308233001Z
**Type:** Feature
**Created:** 2026-03-08T23:30:01Z
**Status:** Draft

## Summary

Instrument crelay with OpenTelemetry distributed tracing to provide per-task token metrics, task timelines, and end-to-end visibility across the conductor workflow. Enables developers to see exactly how long each task took, how many tokens it consumed, and where time was spent.

## Context

crelay orchestrates Claude Code agents to implement conductor tracks. Each track has phases and tasks, each task involves an agent invocation that consumes tokens and time. Today, the `QuotaTracker` records aggregate per-agent costs, but there's no way to see:
- How long each task took (timeline view)
- Per-task token breakdown (not just per-agent aggregate)
- End-to-end flow from track assignment → agent spawn → task completion → PR merge
- Where bottlenecks are (waiting for review? rate limited? large task?)

OpenTelemetry provides the standard for this: traces map to tracks, spans map to phases/tasks, and span attributes carry token metrics.

## Codebase Analysis

- **QuotaTracker** (`adapter/agent/tracker.go`) — records per-agent usage; will coexist with OTel (operational vs observability)
- **StreamEvent parser** (`adapter/agent/parser.go`) — already extracts cost/token data from each Claude Code result event
- **Agent spawner** (`adapter/agent/spawner.go`) — subprocess lifecycle (spawn/monitor/stop) maps to span start/end
- **Webhook handler** (`adapter/rest/server.go`) — Gitea events become span events on the track trace
- **State watcher** (`adapter/dashboard/watcher.go`) — can be augmented to emit span events on state transitions
- **Port/adapter architecture** — OTel should be injected via a port interface for testability

## Prerequisites

- **research-otel-task-tracing_20260308233000Z** must be completed first to determine SDK version, exporter strategy, trace model, and visualization approach

## Acceptance Criteria

- [ ] OTel SDK integrated into crelay with configured trace provider and OTLP exporter
- [ ] Each track implementation produces a trace with spans for phases and tasks
- [ ] Per-task spans include attributes: `tokens.input`, `tokens.output`, `tokens.cache_read`, `cost_usd`, `agent.role`, `agent.id`
- [ ] Agent lifecycle events (spawn, suspend, resume, complete, fail) recorded as span events
- [ ] Webhook events (PR opened, review submitted, etc.) recorded as span events on the relevant trace
- [ ] Task duration is accurately captured (span start = task claimed, span end = task completed)
- [ ] Observability server (Jaeger all-in-one per research) added to docker-compose, starts alongside Gitea
- [ ] Observability UI reverse-proxied through crelay's unified server (e.g., `/-/tracing/`)
- [ ] crelay dashboard links to the observability UI for trace drill-down
- [ ] `crelay init` provisions the observability server alongside Gitea
- [ ] `crelay up`/`down` manages the observability server lifecycle alongside Gitea
- [ ] Dashboard includes a task timeline visualization showing spans with duration and token cost
- [ ] `/-/api/traces` endpoint returns trace data for dashboard consumption (defined in OpenAPI spec)
- [ ] Tracing is **optional and opt-out** — enabled by default but can be disabled:
  - `crelay init --no-tracing` skips the observability server in docker-compose
  - `CRELAY_TRACING_ENABLED=false` env var disables at runtime
  - `crelay config set tracing_enabled false` CLI command to update config
  - Dashboard settings page allows toggling tracing on/off
- [ ] When tracing is disabled: no OTel SDK initialization, no spans emitted, observability server not started
- [ ] When tracing is re-enabled: observability server started, OTel SDK initialized, new spans flow immediately
- [ ] OTel instrumentation has no measurable impact on relay performance
- [ ] QuotaTracker continues to function for rate limiting and budget enforcement (OTel is additive)

## Dependencies

- **research-otel-task-tracing_20260308233000Z** — research track must complete first
- **project-scoped-dashboard_20260308220001Z** — dashboard routing needed for trace timeline view

## Out of Scope

- Instrumenting Claude Code internals (treated as opaque subprocess)
- Cloud-hosted observability (Grafana Cloud, Datadog, etc.) — everything runs locally
- HTTP request-level tracing (middleware instrumentation) — focus is on task-level
- Replacing QuotaTracker — OTel augments, doesn't replace operational quota logic

## Technical Notes

**Proposed trace model (pending research validation):**
```
Trace: track-{trackId}
├── Span: Phase 1 — {phase title}
│   ├── Span: Task 1.1 — {task title}
│   │   ├── Attribute: tokens.input = 45000
│   │   ├── Attribute: tokens.output = 8000
│   │   ├── Attribute: cost_usd = 0.12
│   │   ├── Attribute: agent.id = abc-123
│   │   ├── Attribute: agent.role = developer
│   │   ├── Event: agent_spawned {pid: 12345, worktree: /path}
│   │   ├── Event: webhook_pr_opened {pr: 42}
│   │   └── Event: agent_completed {status: completed}
│   └── Span: Task 1.2 — ...
└── Span: Phase 2 — ...
```

**Port interface for tracing:**
```go
type Tracer interface {
    StartTrackTrace(ctx context.Context, trackID, title string) (context.Context, TrackTrace)
    StartPhaseSpan(ctx context.Context, phase int, title string) (context.Context, PhaseSpan)
    StartTaskSpan(ctx context.Context, task string, title string) (context.Context, TaskSpan)
}
```

**Config additions:**
```json
{
  "tracing_enabled": true
}
```

**CLI config management:**
```
crelay config set tracing_enabled true|false
crelay config get tracing_enabled
```

**Init flag:**
```
crelay init --no-tracing    # skip observability server, set tracing_enabled=false
```

**Env var override:**
```
CRELAY_TRACING_ENABLED=false crelay up   # runtime override
```

**Docker Compose addition:**
```yaml
jaeger:
  image: jaegertracing/all-in-one:latest
  ports:
    - "4318:4318"   # OTLP HTTP receiver
    - "16686:16686" # Jaeger UI
  environment:
    - QUERY_BASE_PATH=/-/tracing
```

**Reverse proxy route in unified server:**
```
/-/tracing/* → jaeger:16686
```

---

_Generated by conductor-track-generator from prompt: "Token metrics per task and task timelines using OpenTelemetry/distributed tracing"_
