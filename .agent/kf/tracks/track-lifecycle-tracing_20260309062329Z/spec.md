# Specification: Track Lifecycle Tracing with OTel

**Track ID:** track-lifecycle-tracing_20260309062329Z
**Type:** Feature
**Created:** 2026-03-09T06:23:29Z
**Status:** Draft

## Summary

Implement a trace-per-track model where one OTel trace spans the entire track lifecycle — from generation through implementation, review, merge, and completion. Link agent session IDs as span attributes so traces can be queried by track ID or session ID.

## Context

The existing OTel infrastructure traces individual agent spawns and webhook events as independent traces. There is no unified trace that follows a track's full lifecycle timeline. The original intent for introducing OTel was to observe the development workflow end-to-end — seeing how long a track takes from creation to completion, where time is spent, and being able to correlate traces with specific Claude Code agent sessions.

### Current OTel State

- **Infrastructure:** Tracer, OTLP exporter, in-memory store, REST API (`/-/api/traces`)
- **Existing spans:** `agent/developer`, `agent/reviewer` (per-spawn), `webhook/{event}` (per-webhook)
- **Gap:** No trace-per-track hierarchy; no session-ID-based trace lookup; no cross-agent trace correlation

### Desired Trace Model

```
trace: track/{trackId}
├── span: track.created                          [generator writes spec]
├── span: track.claimed                          [developer runs implement]
│   ├── span: worktree.acquire                   [pool allocation]
│   └── span: worktree.prepare                   [branch setup]
├── span: agent.developer/{agentId}              [developer agent lifetime]
│   ├── attrs: session.id, agent.pid, worktree
│   ├── event: agent.spawned
│   ├── event: tokens.update {input, output, cache, cost}
│   └── event: agent.completed|failed
├── span: track.pr_created/{prNumber}            [PR opened webhook]
├── span: agent.reviewer/{agentId}               [reviewer agent lifetime]
│   ├── attrs: session.id, pr.number
│   └── event: review.submitted {state}
├── span: track.review_cycle/{cycleNum}          [if changes requested]
│   ├── span: agent.developer.resumed/{agentId}
│   └── span: agent.reviewer/{agentId}
├── span: track.merge                            [PR approved + merged]
│   ├── span: pr.merge
│   ├── span: worktree.return
│   └── event: agents.cleanup
└── span: track.completed                        [final state]
```

Each agent span carries `session.id` as an attribute, enabling lookups like:
- "Show me the trace for track X" → full timeline
- "Show me the trace containing session Y" → find which track an agent session belonged to

## Codebase Analysis

**Existing tracing code:**
- `backend/internal/adapter/tracing/` — OTelTracer, provider, processor, store
- `backend/internal/core/port/tracer.go` — `Tracer` interface with `StartSpan(ctx, name, attrs...)`
- `backend/internal/adapter/agent/spawner.go` — Already creates `agent/developer` and `agent/reviewer` spans
- `backend/internal/adapter/rest/server.go` — Already creates `webhook/{event}` spans

**Key integration points for track-level traces:**
- `backend/internal/adapter/cli/implement.go` — Track claim, worktree acquire, agent spawn
- `backend/internal/adapter/rest/server.go` — PR opened, review submitted, merge triggers
- `backend/internal/core/service/cleanup_service.go` — Merge + cleanup flow
- `backend/internal/core/service/lifecycle_service.go` — Board-driven agent control
- `backend/internal/core/service/pr_service.go` — PR state machine

**Context propagation challenge:**
Track-level traces span multiple processes (CLI → server → agents). The trace context must be persisted with the track so that when the server receives a webhook for a track, it can resume the same trace. Options:
1. **Store trace ID in track metadata** — persist `traceID` alongside `trackID` in board card or a dedicated store
2. **Derive trace ID from track ID** — deterministic mapping (e.g., hash trackID → traceID). Simpler, no storage needed, but less flexible.

**Recommendation:** Option 1 — store trace ID in the board card's metadata. The board card already carries `track_id`, `agent_id`, etc. Adding `trace_id` is natural.

## Acceptance Criteria

- [ ] One OTel trace is created per track when the track is claimed (`implement` command)
- [ ] The trace ID is persisted with the track (board card metadata or dedicated store)
- [ ] Agent spawns create child spans under the track trace (not independent traces)
- [ ] Each agent span includes `session.id` attribute (Claude Code session ID)
- [ ] Webhook-triggered events (PR opened, review submitted, merge) create child spans under the track trace
- [ ] Review cycles are represented as nested spans within the track trace
- [ ] Merge and cleanup create child spans under the track trace
- [ ] REST API `GET /-/api/traces` returns track-level traces with track ID in root span name
- [ ] REST API `GET /-/api/traces/{traceId}` returns the full span tree for a track
- [ ] New REST API endpoint or query param: lookup trace by session ID (`GET /-/api/traces?session_id=X`)
- [ ] New REST API endpoint or query param: lookup trace by track ID (`GET /-/api/traces?track_id=X`)
- [ ] Dashboard trace timeline shows track-level traces with expandable agent/phase spans

## Dependencies

None — the rebrand track has been completed.

## Blockers

None.

## Conflict Risk

None — no other pending tracks.

## Out of Scope

- Jaeger UI integration (already planned separately)
- Network-level HTTP request tracing (not the intent of this instrumentation)
- Persisting traces to disk/SQLite (in-memory store is sufficient for now)
- Trace export to external services (OTLP exporter to local Jaeger is sufficient)
- Conductor skill/generator tracing (track generation happens outside the relay server)

## Technical Notes

### Context Propagation Across Processes

The track trace starts in the CLI (`implement` command) but must continue in the server (webhook handlers). Two approaches:

**Approach A — Trace context in board card:**
1. `implement` command creates trace, stores `traceID` in board card metadata
2. Webhook handlers look up `traceID` from board card by matching `trackID` (derived from PR branch name)
3. Reconstruct parent context using `trace.ContextWithRemoteSpanContext()`

**Approach B — Deterministic trace ID:**
1. Derive trace ID from track ID: `traceID = sha256(trackID)[:16]`
2. Any component can reconstruct the trace context without storage lookup
3. Simpler but couples trace identity to track identity

**Recommendation:** Approach A is cleaner and follows OTel conventions. The board card is already the coordination point for track state.

### Port Interface Extension

The `Tracer` port interface needs extension to support:
- Creating a span with a specific trace ID (for trace context reconstruction)
- Querying traces by attribute value (session ID, track ID)

### Store Extension

The in-memory `Store` needs:
- Index by track ID (root span name or attribute)
- Index by session ID (scan span attributes)
- Both are simple in-memory maps updated on `Record()`

---

_Generated by conductor-track-generator from prompt: "OTel tracing for track lifecycle timeline, linked to agent session IDs"_
