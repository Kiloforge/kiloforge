# Implementation Plan: SSE Event Bus Infrastructure

**Track ID:** sse-event-bus_20260309091500Z

## Phase 1: Event Bus Port & Types

- [x] Task 1.1: Define `EventBus` interface in `backend/internal/core/port/event_bus.go` — `Publish(Event)`, `Subscribe() <-chan Event`, `Unsubscribe(<-chan Event)`, `ClientCount() int`
- [x] Task 1.2: Define `Event` struct in `backend/internal/core/domain/event.go` with `Type string` and `Data any`
- [x] Task 1.3: Add typed event constants and constructor helpers in `backend/internal/core/domain/events.go` — `EventAgentUpdate`, `EventAgentRemoved`, `EventQuotaUpdate`, `EventTrackUpdate`, `EventTrackRemoved`, `EventBoardUpdate`, `EventTraceUpdate`, `EventProjectUpdate`, `EventProjectRemoved`, `EventLockUpdate`, `EventLockReleased`

## Phase 2: Refactor SSEHub to Implement EventBus

- [x] Task 2.1: Refactor `SSEHub` in `backend/internal/adapter/dashboard/sse.go` to implement `port.EventBus` interface using `domain.Event` instead of the local `SSEEvent` type
- [x] Task 2.2: Remove the `SSEEvent` type — replace all usages with `domain.Event`
- [x] Task 2.3: Update `handleSSE` HTTP handler to use `domain.Event` fields for SSE wire format
- [x] Task 2.4: Update watcher to use the `port.EventBus` interface instead of direct `SSEHub` access

## Phase 3: Dependency Injection

- [x] Task 3.1: Add `EventBus port.EventBus` field to `rest.APIHandlerOpts` and `rest.APIHandler`
- [x] Task 3.2: Update dashboard `Server` struct to accept `port.EventBus` instead of owning `SSEHub` internally
- [x] Task 3.3: Update startup wiring (where `Server` and `APIHandler` are constructed) to create a single `SSEHub` and inject it into both

## Phase 4: AsyncAPI Spec & Tests

- [x] Task 4.1: Update `backend/api/asyncapi.yaml` — add message definitions for `track_update`, `track_removed`, `board_update`, `trace_update`, `project_update`, `project_removed`, `lock_update`, `lock_released` with payload schemas
- [x] Task 4.2: Unit tests for `SSEHub` as `EventBus` — publish/subscribe, unsubscribe stops delivery, slow-client non-blocking drop, concurrent access safety
- [x] Task 4.3: Integration test — start SSE HTTP handler, connect EventSource client, publish event via bus, verify client receives it

## Phase 5: Verification

- [x] Task 5.1: Verify `make build` succeeds (backend + frontend embed)
- [x] Task 5.2: Verify `go test ./...` passes
- [x] Task 5.3: Manual verification — start server, open dashboard, confirm existing agent/quota SSE events still work
