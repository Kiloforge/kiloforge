# Implementation Plan: Consolidate Domain Types and Frontend Component Cleanup

**Track ID:** refactor-domain-types-cleanup_20260310041008Z

## Phase 1: Move Domain Types from Service to Domain

### Task 1.1: Move TrackEntry, TrackDetail, ProgressCount to domain/
- Create `domain/track.go` with `TrackEntry`, `TrackDetail`, `ProgressCount` types
- Update `service/track_service.go` to import from domain
- Update all references in adapters (rest, cli, dashboard)

### Task 1.2: Move EscalatedItem to domain/
- Create `domain/escalation.go` with `EscalatedItem` type
- Update `service/agent_service.go` to import from domain
- Update all references

### Task 1.3: Verify backend compiles and tests pass
- `go test ./... -race` passes
- No service layer defines business entity types

### Task 1.4: Verify Phase 1
- `make test` passes

## Phase 2: Frontend Component Cleanup

### Task 2.1: Split ProjectPage into sub-containers
- Extract `BoardContainer` — manages useBoard, KanbanBoard rendering
- Extract `SyncContainer` — manages useOriginSync, SyncPanel rendering
- Extract `AdminContainer` — manages admin operations
- ProjectPage becomes layout/tab router importing sub-containers

### Task 2.2: Fix AgentCard data fetching
- Remove `useTracks()` call from AgentCard
- Add `trackTitle?: string` and `projectSlug?: string` props
- Update parent components (AgentGrid, pages) to pass data via props

### Task 2.3: Verify frontend builds and tests pass
- `npm run build` succeeds (TypeScript check)
- `npm test` passes
- Visual spot check: pages render correctly

### Task 2.4: Verify Phase 2
- `make test` passes (full stack)

## Phase 3: Final Verification

### Task 3.1: Cross-check all changes
- Verify no business entities remain defined in service/
- Verify ProjectPage is under 150 lines
- Verify AgentCard has no hook calls
- Run full test suite
