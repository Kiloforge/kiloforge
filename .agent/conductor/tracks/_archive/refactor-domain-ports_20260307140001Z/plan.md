# Implementation Plan: Extract Domain Types, Port Interfaces, and Service Layer

**Track ID:** refactor-domain-ports_20260307140001Z

## Phase 1: Domain Layer

### Task 1.1: Define sentinel errors
- [x] Create `internal/core/domain/errors.go`
- [x] Define: `ErrProjectNotFound`, `ErrProjectExists`, `ErrAgentNotFound`, `ErrPRTrackingNotFound`, `ErrPoolExhausted`, `ErrGiteaUnreachable`, `ErrForbidden`
- [x] Tests: compile check

### Task 1.2: Extract Project entity
- [x] Move `Project` struct from `internal/project/registry.go` to `internal/core/domain/project.go`
- [x] Add `ProjectStatus` value object
- [x] Update all imports referencing `project.Project` → `domain.Project`
- [x] Tests: build succeeds

### Task 1.3: Extract AgentInfo entity
- [x] Move `AgentInfo` struct from `internal/state/state.go` to `internal/core/domain/agent.go`
- [x] Add `AgentRole`, `AgentStatus` value objects
- [x] Update all imports
- [x] Tests: build succeeds

### Task 1.4: Extract PRTracking entity
- [x] Move `PRTracking` struct from `internal/orchestration/tracking.go` to `internal/core/domain/pr_tracking.go`
- [x] Update all imports
- [x] Tests: build succeeds

### Verification 1
- [x] All domain types in `internal/core/domain/`
- [x] `domain/` has zero imports from `internal/adapter/`
- [x] Build succeeds

## Phase 2: Port Layer

### Task 2.1: Define ProjectStore interface
- [x] Create `internal/core/port/project_store.go`
- [x] Methods: `Get(ctx, slug)`, `List(ctx)`, `Add(ctx, p)`, `FindByRepoName(ctx, name)`, `FindByDir(ctx, dir)`
- [x] Returns domain sentinel errors for not-found

### Task 2.2: Define AgentStore interface
- [x] Create `internal/core/port/agent_store.go`
- [x] Methods: `Load(ctx)`, `Save(ctx)`, `AddAgent(info)`, `FindBySessionID(id)`, `UpdateStatus(id, status)`

### Task 2.3: Define remaining port interfaces
- [x] Move `AgentSpawner` from `adapter/rest/server.go` to `internal/core/port/agent_spawner.go`
- [x] Move `Merger` from orchestration to `internal/core/port/merger.go`
- [x] Move `PoolReturner` from orchestration to `internal/core/port/pool_returner.go`
- [x] Move `GitRunner` from pool to `internal/core/port/git_runner.go`
- [x] Create `internal/core/port/gitea_client.go` interface
- [x] Create `internal/core/port/logger.go` interface
- [x] Create `internal/core/port/doc.go` with not-found convention

### Task 2.4: Update adapters to implement port interfaces
- [x] `project.Registry` implements `port.ProjectStore`
- [x] `state.Store` implements `port.AgentStore`
- [x] `gitea.Client` implements `port.GiteaClient`
- [x] Add compile-time checks: `var _ port.XxxStore = (*AdapterType)(nil)`
- [x] Use domain sentinel errors instead of ad-hoc error strings
- [x] Tests: build succeeds

### Verification 2
- [x] All port interfaces defined
- [x] Adapters implement interfaces with compile-time checks
- [x] Port layer imports only domain
- [x] Build succeeds

## Phase 3: Service Layer

### Task 3.1: Extract PR lifecycle service
- [x] Create `internal/core/service/pr_service.go`
- [x] Extract from `adapter/rest/server.go`: PR tracking creation, review handling, escalation, merge decision
- [x] Service depends on port interfaces only
- [x] Tests: build succeeds

### Task 3.2: Extract track discovery service
- [x] Create `internal/core/service/track_service.go`
- [x] Extract `DiscoverTracks()` logic from `orchestration/tracks.go`
- [x] Accept file content as input (not file path) — I/O stays in adapter
- [x] Tests: build succeeds

### Task 3.3: Extract merge/cleanup service
- [x] Create `internal/core/service/cleanup_service.go`
- [x] Extract `MergeAndCleanup()` from `orchestration/cleanup.go`
- [x] Depends on `port.Merger`, `port.PoolReturner`, `port.AgentStore`
- [x] Tests: build succeeds

### Task 3.4: Update adapter/rest server to use services
- [x] `Server` struct holds service references instead of direct adapter types
- [x] Constructor accepts port interfaces
- [x] HTTP handlers become thin: parse request → call service → write response
- [x] Tests: all existing relay tests pass

### Verification 3
- [x] Service layer imports only `domain/` and `port/`
- [x] `adapter/rest/server.go` is thin HTTP handler layer
- [x] No dependency direction violations
- [x] All tests pass

## Phase 4: Consolidate Persistence

### Task 4.1: Move project persistence to adapter/persistence
- [x] Move `project.Registry` persistence logic to `internal/adapter/persistence/jsonfile/project_store.go`
- [x] Keep domain `Project` in `core/domain/`
- [x] Remove old `internal/project/` package
- [x] Tests: build and test

### Task 4.2: Move state persistence to adapter/persistence
- [x] Move `state.Store` persistence logic to `internal/adapter/persistence/jsonfile/agent_store.go`
- [x] Keep domain `AgentInfo` in `core/domain/`
- [x] Remove old `internal/state/` package
- [x] Tests: build and test

### Task 4.3: Move orchestration persistence to adapter/persistence
- [x] Move `PRTracking` save/load to `internal/adapter/persistence/jsonfile/pr_tracking_store.go`
- [x] Remove old `internal/orchestration/` package
- [x] Tests: build and test

### Verification 4
- [x] `internal/project/`, `internal/state/`, `internal/orchestration/` removed
- [x] All persistence in `internal/adapter/persistence/jsonfile/`
- [x] All tests pass
- [x] Build succeeds
- [x] `go vet ./...` clean
