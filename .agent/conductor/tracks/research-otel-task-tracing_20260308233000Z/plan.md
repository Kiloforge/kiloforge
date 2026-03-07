# Implementation Plan: Research — OpenTelemetry for Task-Level Tracing and Token Metrics

## Phase 1: OTel Go SDK Evaluation (3 tasks)

### Task 1.1: Evaluate OTel Go SDK and exporter options
- Review `go.opentelemetry.io/otel` SDK: stability, API surface, dependencies
- Evaluate exporters: OTLP (gRPC/HTTP), stdout, file-based
- Assess in-process SQLite exporter feasibility (no external collector)
- Document version recommendations and dependency footprint

### Task 1.2: Evaluate local visualization options
- Jaeger all-in-one (Docker container alongside Gitea)
- Grafana Tempo + Grafana (heavier but more dashboarding)
- Custom trace timeline in crelay dashboard (React component)
- Assess trade-offs: setup complexity vs feature richness vs maintenance

### Task 1.3: Assess performance overhead
- Benchmark OTel SDK instrumentation overhead for Go HTTP servers
- Estimate span volume for typical crelay session (5-10 agents, ~50 tasks)
- Document memory and CPU impact

## Phase 2: Trace Model Design (3 tasks)

### Task 2.1: Map conductor concepts to OTel traces and spans
- Define trace boundaries (one trace per track? per implementation cycle?)
- Define span hierarchy: track → phase → task → agent invocation
- Define span attributes: token counts, cost, agent role, status
- Define span events: status transitions, webhook events, quota alerts
- Document the proposed model with examples

### Task 2.2: Evaluate context propagation for subprocess agents
- Research W3C Trace Context propagation via env vars (`TRACEPARENT`)
- Determine if Claude Code respects or passes through env vars
- Fallback: create spans around subprocess boundary (no propagation into agent)
- Document recommended approach

### Task 2.3: Define relationship with existing QuotaTracker
- Map current QuotaTracker responsibilities: rate limiting, budget enforcement, usage aggregation
- Determine what OTel replaces vs augments
- Propose integration: QuotaTracker for operational decisions, OTel for observability
- Document migration path if consolidation is desired

## Phase 3: Decision Document (1 task)

### Task 3.1: Write decision document
- **File:** `.agent/conductor/tracks/research-otel-task-tracing_20260308233000Z/decision.md`
- Summarize findings from all research tasks
- Recommend: SDK version, exporter strategy, trace model, visualization approach
- List trade-offs and alternatives considered
- Provide dependency list and estimated implementation effort
- Include example code snippets for key integration points
