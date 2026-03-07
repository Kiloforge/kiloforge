# Implementation Plan: OpenTelemetry Task-Level Tracing and Token Metrics

## Phase 1: OTel SDK Integration (3 tasks)

### Task 1.1: Add OTel dependencies and trace provider
- Add `go.opentelemetry.io/otel`, SDK, and chosen exporter to `go.mod`
- Create `backend/internal/adapter/tracing/provider.go`
- Initialize `TracerProvider` with configured exporter (per research findings)
- Register as global tracer provider
- Add `tracing_enabled` and `tracing_exporter` to config
- Shutdown hook for flushing spans on relay stop

### Task 1.2: Define tracing port interface
- **File:** `backend/internal/core/port/tracer.go`
- `Tracer` interface with `StartTrackTrace`, `StartPhaseSpan`, `StartTaskSpan` methods
- `TrackTrace`, `PhaseSpan`, `TaskSpan` interfaces with `End()`, `AddEvent()`, `SetAttributes()`
- No-op implementation for when tracing is disabled

### Task 1.3: Implement OTel adapter
- **File:** `backend/internal/adapter/tracing/otel_tracer.go`
- Implement `port.Tracer` using OTel SDK
- Map conductor concepts to OTel spans with correct parent-child relationships
- Set standard attributes: `conductor.track.id`, `conductor.phase`, `conductor.task`, `conductor.agent.id`, `conductor.agent.role`

## Phase 2: Agent Instrumentation (3 tasks)

### Task 2.1: Instrument agent spawner with spans
- **File:** `backend/internal/adapter/agent/spawner.go`
- Start task span when agent is spawned for a task
- Record span events for status transitions (running → waiting → completed)
- End span when agent completes or fails, with appropriate status code

### Task 2.2: Record token metrics as span attributes
- **File:** `backend/internal/adapter/agent/tracker.go`
- On each `RecordEvent`, update the active task span with cumulative token attributes
- Attributes: `tokens.input`, `tokens.output`, `tokens.cache_read`, `tokens.cache_create`, `cost_usd`, `result_count`

### Task 2.3: Instrument webhook events
- **File:** `backend/internal/adapter/rest/server.go`
- On webhook receipt, find the relevant track trace and add span events
- Events: PR opened, review submitted, PR merged, issue state change
- Link webhook spans to track trace via track ID correlation

## Phase 3: Trace API Endpoint (3 tasks)

### Task 3.1: Add traces endpoint to OpenAPI spec
- **File:** `backend/api/openapi.yaml`
- `GET /-/api/traces` — list traces with summary (track ID, duration, total cost, status)
- `GET /-/api/traces/{traceId}` — full trace with span tree, attributes, events
- Define `Trace`, `Span`, `SpanEvent`, `SpanAttributes` schemas
- Run `oapi-codegen` to regenerate

### Task 3.2: Implement trace query handler
- **File:** `backend/internal/adapter/rest/trace_handler.go`
- Implement generated strict handler interface
- Query trace data from exporter backend (OTLP collector API, or local SQLite/file)
- Transform into API response format

### Task 3.3: Test trace API
- **File:** `backend/internal/adapter/rest/trace_handler_test.go`
- Test list traces returns correct summaries
- Test get trace returns full span tree
- Test empty state (no traces)

## Phase 4: Dashboard Timeline View (3 tasks)

### Task 4.1: Create trace timeline component
- **File:** `frontend/src/components/TraceTimeline.tsx`
- Gantt-style visualization showing spans as horizontal bars
- Color-coded by status (running, completed, failed)
- Show duration and token cost on hover/click
- Span hierarchy: phases as groups, tasks as bars within groups

### Task 4.2: Add trace detail page
- **File:** `frontend/src/pages/TracePage.tsx`
- Route: `/-/traces/{traceId}`
- Fetch trace data from `/-/api/traces/{traceId}`
- Render `TraceTimeline` with span details panel
- Show aggregate metrics: total duration, total cost, total tokens

### Task 4.3: Add trace links to dashboard
- **File:** `frontend/src/pages/Dashboard.tsx`
- Add "Timeline" link/icon next to each track in the project view
- Link navigates to `/-/traces/{trackId}`
- Show mini cost/duration badge inline

## Phase 5: Testing and Polish (2 tasks)

### Task 5.1: Integration tests
- Test end-to-end: spawn agent → record events → query trace API → verify span tree
- Test disabled tracing (no-op tracer, no errors)
- Test exporter flush on shutdown

### Task 5.2: Documentation
- Update README with tracing configuration section
- Document trace model and how to view traces
- Add tracing config examples to `config.json` documentation
