# Implementation Plan: Standardize Domain Sentinel Errors and Store Returns

**Track ID:** refactor-error-standardization_20260310041003Z

## Phase 1: Fix Store Sentinel Error Returns

### Task 1.1: Fix agent_store.FindAgent()
- [x] Return `domain.ErrAgentNotFound` when `sql.ErrNoRows`
- [x] Wrap with context: `fmt.Errorf("agent %s: %w", id, domain.ErrAgentNotFound)`
- [x] Update tests to verify `errors.Is(err, domain.ErrAgentNotFound)`

### Task 1.2: Fix project_store.Get()
- [x] Change signature from `(Project, bool)` to `(Project, error)`
- [x] Return `domain.ErrProjectNotFound` when not found
- [x] Update all callers to use `errors.Is()` instead of checking bool

### Task 1.3: Fix pr_tracking_store.LoadPRTracking()
- [x] Return `domain.ErrPRTrackingNotFound` when not found
- [x] Update callers

### Task 1.4: Verify Phase 1
- [x] `go test ./internal/adapter/persistence/sqlite/... -race` passes

## Phase 2: Remove Duplicate Service Error Types

### Task 2.1: Remove ProjectExistsError and ProjectNotFoundError
- [x] Delete custom error struct types from `service/project_service.go`
- [x] Replace with `fmt.Errorf("...: %w", domain.ErrProjectExists)` / `domain.ErrProjectNotFound`
- [x] Update all error creation sites

### Task 2.2: Update REST handler error checking
- [x] Change `errors.As(err, &service.ProjectNotFoundError{})` to `errors.Is(err, domain.ErrProjectNotFound)`
- [x] Change `errors.As(err, &service.ProjectExistsError{})` to `errors.Is(err, domain.ErrProjectExists)`
- [x] Verify HTTP status codes remain correct (404, 409)

### Task 2.3: Update service tests
- [x] Update service layer tests to use `errors.Is()` assertions
- [x] Verify error wrapping preserves context messages

### Task 2.4: Verify Phase 2
- [x] Full test suite passes: `make test`

## Phase 3: Final Verification

### Task 3.1: Verify no custom error types remain in service layer
- [x] Grep for `type.*Error struct` in service/ — should find none
- [x] Verify all `errors.As` calls in REST handler are replaced with `errors.Is`
- [x] Run full test suite with race detector
