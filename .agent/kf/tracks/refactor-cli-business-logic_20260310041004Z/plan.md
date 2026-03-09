# Implementation Plan: Extract Business Logic from CLI Commands to Services

**Track ID:** refactor-cli-business-logic_20260310041004Z

## Phase 1: Extract ImplementService

### Task 1.1: Create ImplementService skeleton [x]
- Create `service/implement_service.go`
- Define constructor with port interface dependencies
- Define method signatures for track validation, preparation, execution

### Task 1.2: Move track validation and consent logic [x]
- Extract track discovery, validation, and consent checking from `cli/implement.go`
- Move to `ImplementService.ValidateTrack()`, `HasConsent()`, `RecordConsent()`
- CLI calls service method, handles user-facing output

### Task 1.3: Move worktree and agent lifecycle logic [x]
- Board state transitions moved to `MoveCardToInProgress()`, `MoveCardToDone()`, `StoreTraceID()`
- Dry-run refactored to use service
- `ListPendingTracks()` added for --list mode

### Task 1.4: Write ImplementService tests [x]
- Test track validation with stub ConsentStore
- Test consent flow with stub ConsentStore
- Test error paths (track not found, already complete, in progress)
- Test ListPendingTracks and LogDir

### Task 1.5: Verify Phase 1 [x]
- `cli/implement.go` delegates business logic to ImplementService
- `go test ./internal/core/service/... -race` passes
- `go test ./internal/adapter/cli/... -race` passes

## Phase 2: Extract SkillsService and Other Logic

### Task 2.1: Create SkillsService [x]
- Create `service/skills_service.go`
- Move update checking, version comparison, installation flow from `cli/skills.go`
- Move config update logic to service
- Created adapter layer in `cli/skills_adapter.go` to bridge config/skills types

### Task 2.2: Extract git sync and SSH key logic [x]
- Created `service/git_sync_service.go` with CheckSyncStatus and PushBranch
- `cli/push.go` delegates sync logic to GitSyncService via execGitRunner adapter
- SSH key discovery already in auth adapter; `cli/add.go` already thin adapter

### Task 2.3: Write tests for extracted services [x]
- SkillsService: test UpdateConfig, CheckForUpdates, InstallUpdate, ListInstalledSkills
- GitSyncService: test CheckSyncStatus (ahead/behind, fetch failure), PushBranch
- All tests use stub/mock implementations of port interfaces

### Task 2.4: Verify Phase 2 [x]
- `cli/skills.go` delegates to SkillsService
- `cli/push.go` delegates to GitSyncService
- `cli/add.go` already thin (delegates to ProjectService)
- `make test` passes — all backend and frontend tests green

## Phase 3: Final Cleanup

### Task 3.1: Verify all CLI commands are thin adapters [x]
- All business logic extracted to service layer
- CLI commands handle: arg parsing, user prompts, output formatting
- `make test` and `make build` both pass
