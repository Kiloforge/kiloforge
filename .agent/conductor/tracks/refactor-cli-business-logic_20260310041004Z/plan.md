# Implementation Plan: Extract Business Logic from CLI Commands to Services

**Track ID:** refactor-cli-business-logic_20260310041004Z

## Phase 1: Extract ImplementService

### Task 1.1: Create ImplementService skeleton
- Create `service/implement_service.go`
- Define constructor with port interface dependencies
- Define method signatures for track validation, preparation, execution

### Task 1.2: Move track validation and consent logic
- Extract track discovery, validation, and consent checking from `cli/implement.go`
- Move to `ImplementService.ValidateAndPrepare()`
- CLI calls service method, handles user-facing output

### Task 1.3: Move worktree and agent lifecycle logic
- Extract worktree acquisition, agent spawning, completion callbacks
- Move to `ImplementService.Execute()`
- Move board state transition logic to service

### Task 1.4: Write ImplementService tests
- Test track validation with mock TrackReader
- Test consent flow with mock ConsentStore
- Test error paths (track not found, consent denied, pool exhausted)

### Task 1.5: Verify Phase 1
- `cli/implement.go` is thin adapter (< 100 lines of logic)
- `go test ./internal/core/service/... -race` passes
- `go test ./internal/adapter/cli/... -race` passes

## Phase 2: Extract SkillsService and Other Logic

### Task 2.1: Create SkillsService
- Create `service/skills_service.go`
- Move update checking, version comparison, installation flow from `cli/skills.go`
- Move config update logic to service

### Task 2.2: Extract git sync and SSH key logic
- Move `cli/push.go:93-143` fetch/ahead-behind to project service or new git sync service
- Move `cli/add.go:199-223` SSH key discovery to auth service or project service
- CLI commands become thin delegates

### Task 2.3: Write tests for extracted services
- SkillsService: test update check, install, config update with mocks
- Git sync: test ahead/behind calculation
- SSH key discovery: test key listing and selection

### Task 2.4: Verify Phase 2
- `cli/skills.go` is thin adapter
- `cli/push.go` and `cli/add.go` are thin adapters
- Full test suite passes: `make test`

## Phase 3: Final Cleanup

### Task 3.1: Verify all CLI commands are thin adapters
- Grep for business logic patterns in cli/ files
- Verify no service-layer concerns remain in CLI
- Run full test suite with race detector
