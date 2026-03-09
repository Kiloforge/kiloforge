# Implementation Plan: E2E Tests — Distributed Tracing

**Track ID:** e2e-distributed-tracing_20260309194839Z

## Phase 1: Trace List Tests

- [ ] Task 1.1: List with seeded traces — seed 3-5 traces with known IDs and durations, navigate to trace list page, verify all traces appear with correct trace ID, duration, and span count
- [ ] Task 1.2: Empty list — start with no traces seeded, navigate to trace list page, verify empty state message is displayed (e.g., "No traces recorded")
- [ ] Task 1.3: Trace metadata display — seed a trace with known root span name and started_at timestamp, verify the list row shows root span name, formatted timestamp, and duration in human-readable format

## Phase 2: Trace Detail Tests

- [ ] Task 2.1: Span hierarchy — seed a trace with 3-level span hierarchy (root -> child -> grandchild), navigate to trace detail, verify all three spans are displayed with correct parent-child indentation or tree structure
- [ ] Task 2.2: Span attributes — seed a trace with spans containing attributes (project slug, track ID, agent ID, webhook event type), navigate to detail, verify attributes are displayed for each span (in a detail panel or inline)
- [ ] Task 2.3: Navigation from list — seed traces, navigate to trace list, click on a trace row, verify URL changes to `/traces/{traceId}` and the detail page loads with the correct trace

## Phase 3: Timeline Tests

- [ ] Task 3.1: Timeline rendering — seed a trace with 3 spans of known durations, navigate to trace detail, verify spans are rendered as visual bars in a timeline view
- [ ] Task 3.2: Relative timing — seed a trace where child span starts 100ms after root span, verify the child span bar is offset from the root span bar in the timeline
- [ ] Task 3.3: Span duration display — seed spans with varying durations (10ms, 500ms, 2000ms), verify the timeline bar widths are proportional to duration, and duration text labels are shown

## Phase 4: Real-Time Tests

- [ ] Task 4.1: trace_update SSE event — open trace list page in Playwright, seed a new trace via API, verify the trace list updates to show the new trace without manual page refresh
- [ ] Task 4.2: New trace appears in list — subscribe to SSE, create a trace, verify `trace_update` event is received AND the UI reflects the new trace in the list
- [ ] Task 4.3: Auto-refresh on update — open trace detail page for an existing trace, add new spans to that trace via API, verify the detail page updates to show the new spans

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Deep nesting — seed a trace with 6-level deep span hierarchy, navigate to detail, verify all levels render without layout overflow or truncation, verify tree/indentation is visible for all levels
- [ ] Task 5.2: Nonexistent trace — navigate directly to `/traces/nonexistent-id`, verify a 404 or "Trace not found" error page is displayed, verify no crash or blank page
- [ ] Task 5.3: Missing spans and API errors — seed a trace with a span referencing a nonexistent parent_span_id, verify the UI handles orphaned spans gracefully (renders them at root level or shows warning); simulate API error during trace list load, verify error state is shown
