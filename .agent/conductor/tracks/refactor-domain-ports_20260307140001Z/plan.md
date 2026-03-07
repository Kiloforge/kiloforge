# Implementation Plan: Extract Domain Types, Port Interfaces, and Service Layer

**Track ID:** refactor-domain-ports_20260307140001Z

## Phase 1: Domain Layer

### Task 1.1: Define sentinel errors
- Create `internal/core/domain/errors.go`
- Define: `ErrProjectNotFound`, `ErrProjectExists`, `ErrAgentNotFound`, `ErrPRTrackingNotFound`, `ErrPoolExhausted`, `ErrGiteaUnreachable`, `ErrForbidden`
- Tests: compile check

### Task 1.2: Extract Project entity
- Move `Project` struct from `internal/project/registry.go` to `internal/core/domain/project.go`
- Add `ProjectStatus` value object
- Update all imports referencing `project.Project` → `domain.Project`
- Tests: build succeeds

### Task 1.3: Extract AgentInfo entity
- Move `AgentInfo` struct from `internal/state/state.go` to `internal/core/domain/agent.go`
- Add `AgentRole`, `AgentStatus` value objects
- Update all imports
- Tests: build succeeds

### Task 1.4: Extract PRTracking entity
- Move `PRTracking` struct from `internal/orchestration/tracking.go` to `internal/core/domain/pr_tracking.go`
- Update all imports
- Tests: build succeeds

### Verification 1
- [ ] All domain types in `internal/core/domain/`
- [ ] `domain/` has zero imports from `internal/adapter/`
- [ ] Build succeeds

## Phase 2: Port Layer

### Task 2.1: Define ProjectStore interface
- Create `internal/core/port/project_store.go`
- Methods: `Get(ctx, slug)`, `List(ctx)`, `Add(ctx, p)`, `FindByRepoName(ctx, name)`, `FindByDir(ctx, dir)`
- Returns domain sentinel errors for not-found

### Task 2.2: Define AgentStore interface
- Create `internal/core/port/agent_store.go`
- Methods: `Load(ctx)`, `Save(ctx)`, `AddAgent(info)`, `FindBySessionID(id)`, `UpdateStatus(id, status)`

### Task 2.3: Define remaining port interfaces
- Move `AgentSpawner` from `adapter/rest/server.go` to `internal/core/port/agent_spawner.go`
- Move `Merger` from orchestration to `internal/core/port/merger.go`
- Move `PoolReturner` from orchestration to `internal/core/port/pool_returner.go`
- Move `GitRunner` from pool to `internal/core/port/git_runner.go`
- Create `internal/core/port/gitea_client.go` interface
- Create `internal/core/port/logger.go` interface
- Create `internal/core/port/doc.go` with not-found convention

### Task 2.4: Update adapters to implement port interfaces
- `project.Registry` implements `port.ProjectStore`
- `state.Store` implements `port.AgentStore`
- `gitea.Client` implements `port.GiteaClient`
- Add compile-time checks: `var _ port.XxxStore = (*AdapterType)(nil)`
- Use domain sentinel errors instead of ad-hoc error strings
- Tests: build succeeds

### Verification 2
- [ ] All port interfaces defined
- [ ] Adapters implement interfaces with compile-time checks
- [ ] Port layer imports only domain
- [ ] Build succeeds

## Phase 3: Service Layer

### Task 3.1: Extract PR lifecycle service
- Create `internal/core/service/pr_service.go`
- Extract from `adapter/rest/server.go`: PR tracking creation, review handling, escalation, merge decision
- Service depends on port interfaces only
- Tests: build succeeds

### Task 3.2: Extract track discovery service
- Create `internal/core/service/track_service.go`
- Extract `DiscoverTracks()` logic from `orchestration/tracks.go`
- Accept file content as input (not file path) — I/O stays in adapter
- Tests: build succeeds

### Task 3.3: Extract merge/cleanup service
- Create `internal/core/service/cleanup_service.go`
- Extract `MergeAndCleanup()` from `orchestration/cleanup.go`
- Depends on `port.Merger`, `port.PoolReturner`, `port.AgentStore`
- Tests: build succeeds

### Task 3.4: Update adapter/rest server to use services
- `Server` struct holds service references instead of direct adapter types
- Constructor accepts port interfaces
- HTTP handlers become thin: parse request → call service → write response
- Tests: all existing relay tests pass

### Verification 3
- [ ] Service layer imports only `domain/` and `port/`
- [ ] `adapter/rest/server.go` is thin HTTP handler layer
- [ ] No dependency direction violations
- [ ] All tests pass

## Phase 4: Consolidate Persistence

### Task 4.1: Move project persistence to adapter/persistence
- Move `project.Registry` persistence logic to `internal/adapter/persistence/jsonfile/project_store.go`
- Keep domain `Project` in `core/domain/`
- Remove old `internal/project/` package
- Tests: build and test

### Task 4.2: Move state persistence to adapter/persistence
- Move `state.Store` persistence logic to `internal/adapter/persistence/jsonfile/agent_store.go`
- Keep domain `AgentInfo` in `core/domain/`
- Remove old `internal/state/` package
- Tests: build and test

### Task 4.3: Move orchestration persistence to adapter/persistence
- Move `PRTracking` save/load to `internal/adapter/persistence/jsonfile/pr_tracking_store.go`
- Remove old `internal/orchestration/` package
- Tests: build and test

### Verification 4
- [ ] `internal/project/`, `internal/state/`, `internal/orchestration/` removed
- [ ] All persistence in `internal/adapter/persistence/jsonfile/`
- [ ] All tests pass
- [ ] Build succeeds
- [ ] `go vet ./...` clean
