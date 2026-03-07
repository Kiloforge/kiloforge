# Implementation Plan: Test Coverage Alignment with Style Guide

**Track ID:** test-coverage-alignment_20260307140002Z

## Phase 1: Shared Test Infrastructure

### Task 1.1: Create testutil package with mock stores
- Create `internal/core/testutil/mocks.go`
- Implement: `MockProjectStore`, `MockAgentStore`, `MockLogger`
- All mocks satisfy port interfaces with in-memory maps
- Return domain sentinel errors for not-found cases
- Tests: mocks compile and satisfy interfaces

### Task 1.2: Create testutil mock clients
- Add `MockGiteaClient`, `MockAgentSpawner`, `MockMerger`, `MockPoolReturner`
- Configurable behavior: success by default, injectable errors
- Tests: mocks compile and satisfy interfaces

### Verification 1
- [ ] `internal/core/testutil/` package exists
- [ ] All mocks satisfy their port interfaces (compile-time checks)
- [ ] Build succeeds

## Phase 2: Domain Layer Tests

### Task 2.1: Test domain entities and value objects
- Test `Project`, `AgentInfo`, `PRTracking` structs
- Test value object validation (status transitions, slug formats)
- Test sentinel errors are distinct and checkable with `errors.Is()`
- Table-driven tests for all validation

### Task 2.2: Test authorization (if implemented)
- Test `HasPermission()` for all role × activity combinations
- Test `Authorize()` returns `ErrForbidden` correctly
- Test `AuthzRegistry` operation→activity mapping
- Table-driven tests

### Verification 2
- [ ] Domain types have comprehensive validation tests
- [ ] Sentinel errors tested
- [ ] All tests use `t.Parallel()`

## Phase 3: Service Layer Tests

### Task 3.1: Test PR service
- Happy path: create tracking, handle review, approve, merge
- Error paths: project not found, agent spawn failure, Gitea API error
- Escalation path: max review cycles exceeded
- Use testutil mocks for all dependencies

### Task 3.2: Test track discovery service
- Parse valid tracks.md content
- Handle malformed markdown gracefully
- Filter pending tracks only
- Table-driven tests for various markdown formats

### Task 3.3: Test cleanup service
- Happy path: merge → comment → delete branch → return worktree → update state
- Error paths: merge conflict, pool return failure, state save failure
- Verify partial failure doesn't leave inconsistent state

### Verification 3
- [ ] Service tests use only testutil mocks (no adapter imports)
- [ ] Error propagation tested for each dependency failure
- [ ] All tests use `t.Parallel()`

## Phase 4: Adapter Layer Tests

### Task 4.1: Test persistence adapters
- JSON project store: save/load round-trip, not-found, duplicate, corrupt JSON, empty file
- JSON agent store: save/load round-trip, not-found, corrupt data
- JSON PR tracking store: save/load, concurrent access
- Use `t.TempDir()` for filesystem isolation
- Table-driven tests for edge cases

### Task 4.2: Test REST handlers
- Webhook dispatch: each event type routes correctly
- Error responses: correct HTTP status codes for domain errors
- Internal error leakage: verify no DB paths, stack traces, etc. in responses
- Use `httptest.NewRecorder()` and testutil service mocks

### Task 4.3: Test agent spawner
- Successful spawn returns agent info
- Spawn failure returns error
- Process lifecycle tracking

### Task 4.4: Test Gitea client (fill gaps)
- Test all client methods (currently only 8 of 15+ tested)
- Error responses from API (4xx, 5xx)
- Use `httptest.NewServer()` for HTTP mocking
- Table-driven tests for API responses

### Task 4.5: Migrate scattered mocks to testutil
- Replace local `fakeSpawner` in rest/server_test.go with `testutil.MockAgentSpawner`
- Replace local `mockMerger`/`mockPoolReturner` in orchestration tests
- Replace local `fakeGitRunner` in pool tests
- Verify all tests still pass after migration

### Verification 4
- [ ] Every adapter package has tests
- [ ] Error paths tested for each adapter
- [ ] REST handler tests check for error detail leakage
- [ ] Scattered mocks replaced with shared testutil
- [ ] All tests pass with `-race` flag
- [ ] All tests use `t.Parallel()` where safe
