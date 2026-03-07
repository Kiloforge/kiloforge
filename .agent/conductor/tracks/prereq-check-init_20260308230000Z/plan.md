# Implementation Plan: Prerequisite Check During Init

## Phase 1: Prereq Checker (4 tasks)

### Task 1.1: Create prereq package with Check function
- [x] **File:** `backend/internal/adapter/prereq/check.go`

### Task 1.2: Add platform-specific install hints
- [x] Integrated into `check.go` (darwin/linux hints for git, docker, compose, claude)

### Task 1.3: Add formatting for prereq errors
- [x] `FormatErrors()` shows all missing tools at once with install hints

### Task 1.4: Write tests for prereq checker
- [x] **File:** `backend/internal/adapter/prereq/check_test.go`
- Tests for FormatErrors, platform hints, Check on dev machine

## Phase 2: Wire Into Init (2 tasks)

### Task 2.1: Call prereq check at top of runInit
- [x] **File:** `backend/internal/adapter/cli/init.go`
- prereq.Check() called before config resolution

### Task 2.2: Verify init still works end-to-end
- [x] All tests pass, no extra output when tools present

## Phase 3: Tests and Verification (2 tasks)

### Task 3.1: Add smoke test for prereq check
- [x] TestCheck_AllPresent in check_test.go

### Task 3.2: Run full test suite
- [x] `make test` — all pass
