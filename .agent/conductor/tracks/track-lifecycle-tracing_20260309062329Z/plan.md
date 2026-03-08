# Implementation Plan: Track Lifecycle Tracing with OTel

**Track ID:** track-lifecycle-tracing_20260309062329Z

## Phase 1: Trace Context Storage & Propagation

- [ ] Task 1.1: Add `trace_id` field to `domain.BoardCard` struct
- [ ] Task 1.2: Add `TraceContextStore` port interface — `StoreTraceID(trackID, traceID)`, `GetTraceID(trackID) (traceID, bool)`
- [ ] Task 1.3: Implement `TraceContextStore` backed by board card metadata (read/write `trace_id` field)
- [ ] Task 1.4: Extend `Tracer` port with `StartSpanWithTraceID(ctx, traceID, name, attrs...) (ctx, SpanEnder)` for reconstructing trace context
- [ ] Task 1.5: Implement `StartSpanWithTraceID` in `OTelTracer` — create remote span context from stored trace ID
- [ ] Task 1.6: Tests for trace context storage and reconstruction

## Phase 2: Track Claim Tracing (implement command)

- [ ] Task 2.1: Create root trace span `track/{trackId}` in `implement` command when claiming a track
- [ ] Task 2.2: Store the trace ID in board card via `TraceContextStore`
- [ ] Task 2.3: Create child spans for `worktree.acquire` and `worktree.prepare`
- [ ] Task 2.4: Pass trace context to agent spawner so developer agent span is a child of the track trace
- [ ] Task 2.5: Add `session.id` attribute to agent spans in spawner
- [ ] Task 2.6: Tests for implement command tracing — verify trace hierarchy and attributes

## Phase 3: Webhook-Driven Trace Continuation

- [ ] Task 3.1: In webhook handler, extract `trackID` from PR branch name
- [ ] Task 3.2: Look up `traceID` from `TraceContextStore` using `trackID`
- [ ] Task 3.3: Reconstruct trace context and create child span `track.pr_created/{prNumber}` under the track trace
- [ ] Task 3.4: Create child span `agent.reviewer/{agentId}` under the track trace (not independent trace)
- [ ] Task 3.5: Add `session.id` attribute to reviewer agent span
- [ ] Task 3.6: Create child span `track.review_cycle/{cycleNum}` for changes-requested → re-review flows
- [ ] Task 3.7: Tests for webhook trace continuation — verify spans join the correct trace

## Phase 4: Merge & Completion Tracing

- [ ] Task 4.1: Create child span `track.merge` under the track trace when PR is approved and merged
- [ ] Task 4.2: Create nested spans for `pr.merge`, `worktree.return`, and `agents.cleanup`
- [ ] Task 4.3: Add event `track.completed` to the root track span when merge + cleanup finishes
- [ ] Task 4.4: End the root track span after completion
- [ ] Task 4.5: Tests for merge tracing — verify full span tree from claim to completion

## Phase 5: Store Indexing & Query API

- [ ] Task 5.1: Add secondary indexes to in-memory `Store` — index by `track.id` attribute and `session.id` attribute
- [ ] Task 5.2: Update `Store.Record()` to populate indexes on span ingestion
- [ ] Task 5.3: Add `Store.FindByTrackID(trackID)` and `Store.FindBySessionID(sessionID)` methods
- [ ] Task 5.4: Update OpenAPI spec — add `track_id` and `session_id` query params to `GET /-/api/traces`
- [ ] Task 5.5: Regenerate API code (`make gen-api`)
- [ ] Task 5.6: Implement query param handling in `ListTraces()` handler
- [ ] Task 5.7: Tests for indexed queries — verify lookup by track ID and session ID

## Phase 6: Dashboard Integration

- [ ] Task 6.1: Add track ID column/link to trace list in dashboard
- [ ] Task 6.2: Add "View Trace" action to board cards that have a `trace_id`
- [ ] Task 6.3: Trace detail view shows expandable span tree with session IDs as clickable metadata
- [ ] Task 6.4: Verify dashboard renders track-level traces correctly

## Phase 7: Final Verification

- [ ] Task 7.1: Run `make build` — compiles cleanly
- [ ] Task 7.2: Run `make test` — all tests pass
- [ ] Task 7.3: Run `make lint` — no lint errors
- [ ] Task 7.4: Manual verification: claim a track, observe trace creation, verify webhook spans join the trace
- [ ] Task 7.5: Verify `GET /-/api/traces?track_id=X` returns the correct trace
- [ ] Task 7.6: Verify `GET /-/api/traces?session_id=Y` returns the trace containing that session
