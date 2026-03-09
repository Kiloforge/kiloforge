# Implementation Plan: Standardize Domain Sentinel Errors and Store Returns

**Track ID:** refactor-error-standardization_20260310041003Z

## Phase 1: Fix Store Sentinel Error Returns

### Task 1.1: Fix agent_store.FindAgent()
- Return `domain.ErrAgentNotFound` when `sql.ErrNoRows`
- Wrap with context: `fmt.Errorf("agent %s: %w", id, domain.ErrAgentNotFound)`
- Update tests to verify `errors.Is(err, domain.ErrAgentNotFound)`

### Task 1.2: Fix project_store.Get()
- Change signature from `(Project, bool, error)` to `(Project, error)` if feasible
- Or: return `domain.ErrProjectNotFound` when not found
- Update all callers to use `errors.Is()` instead of checking bool

### Task 1.3: Fix pr_tracking_store.LoadPRTracking()
- Return `domain.ErrPRTrackingNotFound` when not found
- Update callers

### Task 1.4: Verify Phase 1
- `go test ./internal/adapter/persistence/sqlite/... -race` passes

## Phase 2: Remove Duplicate Service Error Types

### Task 2.1: Remove ProjectExistsError and ProjectNotFoundError
- Delete custom error struct types from `service/project_service.go`
- Replace with `fmt.Errorf("...: %w", domain.ErrProjectExists)` / `domain.ErrProjectNotFound`
- Update all error creation sites

### Task 2.2: Update REST handler error checking
- Change `errors.As(err, &service.ProjectNotFoundError{})` to `errors.Is(err, domain.ErrProjectNotFound)`
- Change `errors.As(err, &service.ProjectExistsError{})` to `errors.Is(err, domain.ErrProjectExists)`
- Verify HTTP status codes remain correct (404, 409)

### Task 2.3: Update service tests
- Update service layer tests to use `errors.Is()` assertions
- Verify error wrapping preserves context messages

### Task 2.4: Verify Phase 2
- Full test suite passes: `make test`

## Phase 3: Final Verification

### Task 3.1: Verify no custom error types remain in service layer
- Grep for `type.*Error struct` in service/ — should find none
- Verify all `errors.As` calls in REST handler are replaced with `errors.Is`
- Run full test suite with race detector
