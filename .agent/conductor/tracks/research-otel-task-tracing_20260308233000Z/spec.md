# Specification: Research — OpenTelemetry for Task-Level Tracing and Token Metrics

**Track ID:** research-otel-task-tracing_20260308233000Z
**Type:** Research
**Created:** 2026-03-08T23:30:00Z
**Status:** Draft

## Summary

Investigate how OpenTelemetry (OTel) distributed tracing can be integrated into crelay to provide per-task token metrics, task timelines, and end-to-end visibility into the conductor workflow — from track assignment through agent execution to PR merge.

## Context

crelay already tracks per-agent token usage (cost, input/output/cache tokens) via the `QuotaTracker`, and broadcasts state changes via SSE. However, there's no timeline view showing how long each task took, no per-task (as opposed to per-agent) cost breakdown, and no structured way to correlate events across the relay → agent → webhook pipeline. OpenTelemetry's distributed tracing model (traces → spans → attributes) maps naturally onto conductor's workflow: a track is a trace, each task/phase is a span, and token usage is span attributes.

## Codebase Analysis

- **QuotaTracker** (`adapter/agent/tracker.go`) — already parses stream-json events from Claude Code and records per-agent `AgentUsage` (cost, tokens, result count). This is the natural place to emit OTel spans for each "result" event.
- **StreamEvent parser** (`adapter/agent/parser.go`) — extracts `CostUSD`, `Usage.InputTokens`, `Usage.OutputTokens`, etc. from each line. These become span attributes.
- **Agent lifecycle** (`adapter/agent/spawner.go`, `recovery.go`, `shutdown.go`) — spawn/suspend/resume/kill transitions map to span start/end events with status codes.
- **Webhook relay** (`adapter/rest/server.go:handleWebhook`) — Gitea events (issue created, PR opened, review submitted) are natural span events on the track trace.
- **SSE watcher** (`adapter/dashboard/watcher.go`) — polls state every 2s. OTel could replace or augment this with proper event propagation.
- **No existing OTel dependency** — tech stack has no tracing library. This is greenfield.
- **AsyncAPI schema** (`api/asyncapi.yaml`) — documents SSE and webhook event schemas. OTel would add a third observability channel.
- **Logger interface** (`core/port/logger.go`) — simple `Printf`. Would need to coexist with OTel's structured approach.

## Research Questions

1. **OTel SDK for Go** — What's the current state of `go.opentelemetry.io/otel`? What exporters are suitable for a local-first tool (OTLP to local collector, or direct file/SQLite export)?
2. **Trace model mapping** — How should conductor concepts map to OTel?
   - Track → Trace (single trace per track lifecycle)
   - Phase → Parent span
   - Task → Child span (with token metrics as attributes)
   - Agent spawn/stop → Span events
   - Webhook events → Span events or linked spans
3. **Context propagation** — crelay spawns `claude` as a subprocess. Can we pass trace context via env vars or command-line args? Or do we treat the agent as an opaque black box and create spans around the subprocess boundary?
4. **Token metrics as OTel metrics vs span attributes** — Should per-task token counts be OTel Metrics (counters/histograms for aggregation) or span attributes (for per-invocation visibility)? Or both?
5. **Local collector vs embedded** — For a local dev tool, is running an OTel Collector (e.g., via Docker alongside Gitea) acceptable? Or should we use an in-process exporter (e.g., export spans to SQLite or JSON)?
6. **Visualization** — What's the lightest way to visualize traces? Jaeger all-in-one via Docker? Grafana Tempo? A custom trace timeline view in the dashboard?
7. **Overhead** — What's the performance impact of OTel instrumentation on a relay that spawns ~5-10 agents?
8. **Integration with existing quota tracking** — Can OTel replace `QuotaTracker`, or should they coexist? The tracker serves rate-limiting/budget enforcement, which is operational — OTel serves observability, which is informational.

## Acceptance Criteria

- [ ] Document recommended OTel Go SDK version and dependencies
- [ ] Propose trace model: how tracks, phases, tasks, agents, and events map to traces/spans
- [ ] Evaluate context propagation options for subprocess agents
- [ ] Recommend exporter strategy (local collector vs embedded)
- [ ] Recommend visualization approach (existing tool vs custom dashboard)
- [ ] Assess performance overhead for crelay's scale
- [ ] Determine relationship between OTel and existing QuotaTracker
- [ ] Produce a decision document with recommended approach and trade-offs

## Dependencies

- No internal track dependencies
- External: familiarity with OpenTelemetry Go SDK

## Out of Scope

- Actual implementation (separate track)
- Instrumenting Claude Code itself (black box)
- Production-grade OTel infrastructure (this is a local dev tool)

## Technical Notes

**Key OTel Go packages to evaluate:**
- `go.opentelemetry.io/otel` — API
- `go.opentelemetry.io/otel/sdk/trace` — SDK
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace` — OTLP exporter
- `go.opentelemetry.io/otel/exporters/stdout/stdouttrace` — stdout/file exporter

**Potential trace structure:**
```
Track: skill-install-update_20260308231000Z
├── Phase 1: Config and GitHub Client
│   ├── Task 1.1: Add skills config fields [span: 12m, tokens: 45k in / 8k out, cost: $0.12]
│   ├── Task 1.2: Create GitHub release checker [span: 18m, tokens: 62k in / 12k out, cost: $0.18]
│   └── Task 1.3: Test GitHub release checker [span: 8m, tokens: 30k in / 5k out, cost: $0.08]
├── Phase 2: Skill Installer
│   └── ...
```

**Event types that become span events:**
- Agent spawned (with PID, worktree, session ID)
- Agent status change (running → waiting → completed)
- Webhook received (PR opened, review submitted)
- Quota threshold crossed (rate limited, budget warning)

---

_Generated by conductor-track-generator from prompt: "Token metrics per task and task timelines using OpenTelemetry/distributed tracing"_
