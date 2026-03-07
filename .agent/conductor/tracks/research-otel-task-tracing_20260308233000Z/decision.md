# Decision: OpenTelemetry for Task-Level Tracing and Token Metrics

**Track:** research-otel-task-tracing_20260308233000Z
**Date:** 2026-03-08
**Status:** Accepted

## Executive Summary

Integrate OpenTelemetry distributed tracing into crelay to provide per-task token metrics, task timelines, and end-to-end visibility. Use the stable Go SDK with Jaeger all-in-one for visualization, keeping the existing QuotaTracker for operational decisions.

---

## 1. OTel Go SDK Evaluation

### Recommended Version

- **API:** `go.opentelemetry.io/otel` v1.31+ (stable traces, metrics, logs)
- **SDK:** `go.opentelemetry.io/otel/sdk` v1.31+
- **OTLP exporter:** `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`

The Go SDK traces API has been stable since 2022. Metrics SDK reached stable in 2024. The full signal set (traces + metrics + logs) is now production-ready.

### Dependency Footprint

Adding OTel introduces ~5 direct dependencies. The SDK is modular — import only what you use. No CGO required. Compatible with Go 1.22+.

### Exporter Strategy: OTLP to Jaeger All-in-One

**Recommended:** OTLP HTTP exporter → Jaeger all-in-one (Docker container).

**Alternatives considered:**

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| OTLP → Jaeger all-in-one | Zero config, OTLP native, in-memory OK for dev | Another Docker container | **Selected** |
| SQLite exporter (wperron/sqliteexporter) | No external process, fully local | Alpha quality, custom queries needed, no UI | Rejected |
| stdout/file exporter | Simplest, no dependencies | No visualization, JSON flood | Rejected — debug only |
| Grafana Tempo + Grafana | Production-grade dashboarding | Heavy for a local dev tool (2+ containers) | Rejected — overkill |

**Rationale:** Jaeger all-in-one is a single Docker container (already running Docker for Gitea), accepts OTLP natively, provides a full trace UI, and uses in-memory storage by default — perfect for a local dev tool where traces don't need to survive restarts. It can be added to the existing docker-compose.yml as an optional service.

---

## 2. Trace Model

### Mapping Conductor Concepts to OTel

```
Trace: one per track implementation lifecycle
│
├── Span: "track/{trackId}"  (root span, entire track duration)
│   ├── Attributes: track_id, track_type, title, worker
│   │
│   ├── Span: "phase/1"  (phase duration)
│   │   ├── Attributes: phase_number, task_count
│   │   │
│   │   ├── Span: "task/1.1"  (task duration)
│   │   │   ├── Attributes: input_tokens, output_tokens, cache_read_tokens,
│   │   │   │   cache_creation_tokens, cost_usd, result_count
│   │   │   └── Events: "agent.spawned", "agent.completed"
│   │   │
│   │   └── Span: "task/1.2"  ...
│   │
│   ├── Span: "phase/2"  ...
│   │
│   └── Span: "merge"  (merge sequence duration)
│       └── Events: "lock.acquired", "rebase.completed", "merge.completed"
```

### Span Attributes for Token Metrics

Each task span carries these attributes:

```go
span.SetAttributes(
    attribute.Int("tokens.input", usage.InputTokens),
    attribute.Int("tokens.output", usage.OutputTokens),
    attribute.Int("tokens.cache_read", usage.CacheReadTokens),
    attribute.Int("tokens.cache_creation", usage.CacheCreationTokens),
    attribute.Float64("cost.usd", usage.TotalCostUSD),
    attribute.Int("result.count", usage.ResultCount),
    attribute.String("agent.id", agentID),
)
```

### Span Events

Status transitions and external events become span events:

```go
span.AddEvent("agent.spawned", trace.WithAttributes(
    attribute.Int("pid", pid),
    attribute.String("worktree", worktreePath),
))
span.AddEvent("webhook.received", trace.WithAttributes(
    attribute.String("event_type", "pull_request"),
    attribute.String("action", "opened"),
))
```

---

## 3. Context Propagation for Subprocess Agents

### Recommendation: Black-Box Boundary Spans

Claude Code is an opaque subprocess. We **cannot** propagate trace context into it because:
1. `claude` CLI doesn't support W3C `TRACEPARENT` env var passthrough
2. Even if it did, we can't control internal span creation
3. The stream-json output already gives us everything we need (tokens, cost, events)

**Approach:** Create a span around the subprocess lifecycle. Parse stream-json events as they arrive and record them as span events/attributes. The span starts when the agent is spawned and ends when the process exits.

```go
ctx, span := tracer.Start(ctx, "agent/invoke",
    trace.WithAttributes(attribute.String("agent.id", agentID)))
defer span.End()

// ... spawn claude subprocess ...
// ... parse stream-json events, update span attributes ...
```

This gives full visibility into agent duration and token consumption without needing to instrument Claude Code itself.

---

## 4. Relationship with QuotaTracker

### Recommendation: Coexist, Don't Replace

| Concern | QuotaTracker | OTel |
|---------|-------------|------|
| Rate limiting | Yes — `IsRateLimited()`, `RetryAfter()` | No |
| Budget enforcement | Yes — `MaxSessionCostUSD` check | No |
| Per-agent usage aggregation | Yes — `GetAgentUsage()` | Yes (via span attributes) |
| Timeline visualization | No | Yes |
| Per-task cost breakdown | No | Yes |
| Cross-agent correlation | No | Yes (via traces) |
| Persistence | JSON file | In-memory (Jaeger) or OTLP export |

**Decision:** QuotaTracker stays for operational decisions (rate limiting, budget enforcement). OTel adds observability on top. The `RecordEvent()` method in QuotaTracker is the natural integration point — after recording usage, also update the current OTel span.

**Migration path:** If OTel proves sufficient for aggregation queries, QuotaTracker's reporting methods (`GetAgentUsage`, `GetTotalUsage`) could eventually be replaced by Jaeger queries, but the rate-limiting logic must remain as an in-process check.

---

## 5. Visualization

### Recommendation: Jaeger All-in-One + Future Dashboard Widget

**Phase 1 (implementation track):** Add Jaeger all-in-one as an optional service in docker-compose.yml. Port 16686 for UI. Traces flow via OTLP HTTP on port 4318.

```yaml
jaeger:
  image: jaegertracing/all-in-one:1.76
  ports:
    - "16686:16686"   # Jaeger UI
    - "4318:4318"     # OTLP HTTP
  environment:
    COLLECTOR_OTLP_ENABLED: "true"
```

**Phase 2 (future):** Add a "Task Timeline" widget to the React dashboard that queries Jaeger's API (`/api/traces`) and renders a Gantt-style view per track.

---

## 6. Performance Assessment

For crelay's scale (5-10 agents, ~50 tasks per session):

- **Span volume:** ~100-200 spans per session (trivial)
- **Memory overhead:** OTel SDK batches spans in memory, ~1KB per span → <1MB total
- **CPU overhead:** Negligible — span creation is <1μs, batch export is async
- **Network:** OTLP HTTP batch export every 5s — a few KB per batch

**Conclusion:** Zero performance concern at this scale. OTel is designed for high-throughput services processing millions of spans. crelay's workload is orders of magnitude below that.

---

## 7. Implementation Approach

### Integration Points

1. **`adapter/agent/spawner.go`** — Start/end spans around agent subprocess lifecycle
2. **`adapter/agent/tracker.go:RecordEvent()`** — Update current span attributes with token metrics
3. **`adapter/rest/server.go:handleWebhook()`** — Add span events for webhook receipts
4. **`adapter/cli/init.go`** — Initialize OTel tracer provider, configure OTLP exporter
5. **`adapter/compose/template.go`** — Add optional Jaeger service to compose file

### Example: Instrumenting RecordEvent

```go
func (t *QuotaTracker) RecordEvent(ctx context.Context, agentID string, event StreamEvent) {
    // Existing logic unchanged...

    // OTel integration: update span attributes
    span := trace.SpanFromContext(ctx)
    if span.IsRecording() && event.Usage != nil {
        span.SetAttributes(
            attribute.Int("tokens.input", usage.InputTokens),
            attribute.Int("tokens.output", usage.OutputTokens),
            attribute.Float64("cost.usd", usage.TotalCostUSD),
        )
    }
}
```

### Estimated Implementation Effort

- **New package:** `adapter/otel/` — tracer provider setup, config (~100 LOC)
- **Modify:** `spawner.go`, `tracker.go`, `server.go`, `init.go` (~150 LOC changes)
- **Compose:** Add Jaeger service (~10 lines YAML)
- **Tests:** Span creation/attribute verification (~100 LOC)
- **Total:** ~360 LOC, 1 implementation track

### Dependencies to Add

```
go.opentelemetry.io/otel v1.31+
go.opentelemetry.io/otel/sdk v1.31+
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
```

---

## 8. Trade-offs

| Decision | Trade-off |
|----------|-----------|
| Jaeger over SQLite | Requires Docker container but provides rich UI for free |
| Black-box agent spans | No visibility inside Claude Code, but stream-json gives us tokens/cost |
| Coexist with QuotaTracker | Some data duplication, but clean separation of concerns |
| Optional Jaeger | Tracing gracefully degrades to no-op if Jaeger isn't running |
| In-memory storage | Traces lost on restart, but acceptable for local dev tool |

---

## Sources

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [OpenTelemetry Go Releases](https://github.com/open-telemetry/opentelemetry-go/releases)
- [SQLite Exporter for OTel](https://github.com/wperron/sqliteexporter)
- [OTel File Exporter Spec](https://opentelemetry.io/docs/specs/otel/protocol/file-exporter/)
- [Jaeger All-in-One Docker](https://hub.docker.com/r/jaegertracing/all-in-one)
- [Jaeger Getting Started](https://www.jaegertracing.io/docs/1.76/getting-started/)
- [Setting Up Jaeger as OTel Backend](https://oneuptime.com/blog/post/2026-02-06-jaeger-trace-backend-opentelemetry/view)
