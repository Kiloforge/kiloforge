# Implementation Plan: Consolidate Domain Types and Frontend Component Cleanup

**Track ID:** refactor-domain-types-cleanup_20260310041008Z

## Phase 1: Move Domain Types from Service to Domain

### Task 1.1: Move TrackEntry, TrackDetail, ProgressCount to domain/
- [x] Create `domain/track.go` with `TrackEntry`, `TrackDetail`, `ProgressCount` types
- [x] Update `port/track_reader.go` to re-export via type aliases
- [x] All references continue to work via backward-compatible aliases

### Task 1.2: Move EscalatedItem to domain/
- [x] Create `domain/escalation.go` with `EscalatedItem` type
- [x] Update `service/agent_service.go` to import from domain
- [x] All references updated

### Task 1.3: Verify backend compiles and tests pass
- [x] `go build ./...` passes
- [x] No service layer defines business entity types

### Task 1.4: Verify Phase 1
- [x] Backend compiles clean

## Phase 2: Frontend Component Cleanup

### Task 2.1: Split ProjectPage into sub-containers
- [x] Extract `BoardContainer` — manages useBoard, KanbanBoard rendering
- [x] Extract `SyncContainer` — manages useOriginSync, SyncPanel rendering
- [x] Extract `AdminContainer` — manages admin operations
- [x] ProjectPage reduced from 361 to 197 lines

### Task 2.2: Fix AgentCard data fetching
- [x] Remove `useTracks()` call from AgentCard
- [x] Add `projectSlug?: string | null` prop
- [x] Update AgentGrid to compute track-to-project mapping and pass via props
- [x] Update OverviewPage to pass tracks to AgentGrid

### Task 2.3: Verify frontend builds and tests pass
- [x] `tsc -b --noEmit` succeeds
- [x] `npm test` passes (139 tests)

### Task 2.4: Verify Phase 2
- [x] `make build` passes (full stack)

## Phase 3: Final Verification

### Task 3.1: Cross-check all changes
- [x] No business entities remain defined in service/ (EscalatedItem moved)
- [x] ProjectPage reduced from 361 to 197 lines
- [x] AgentCard has no data-fetching hook calls
- [x] Full build passes
