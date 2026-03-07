# Implementation Plan: Prerequisite Check During Init

## Phase 1: Prereq Checker (4 tasks)

### Task 1.1: Create prereq package with Check function
- **File:** `backend/internal/adapter/prereq/check.go`
- Define `PrereqError` struct: `Tool string`, `Reason string`, `InstallHint string`
- Implement `Check() []PrereqError` — checks git, docker, docker compose, claude
- Use `exec.LookPath()` for tool presence
- For docker compose, reuse detection logic from `compose.Detect()` or check both variants

### Task 1.2: Add platform-specific install hints
- **File:** `backend/internal/adapter/prereq/hints.go`
- Use `runtime.GOOS` to select platform (darwin, linux)
- Return appropriate install instructions per tool per platform
- Include URLs for Docker Desktop and Claude Code

### Task 1.3: Add formatting for prereq errors
- **File:** `backend/internal/adapter/prereq/check.go`
- `FormatErrors(errs []PrereqError) string` — formats all errors into a user-friendly message
- Show all missing tools at once, each with install hint
- Example output:
  ```
  Missing prerequisites:

    git — required for repository management
      Install: xcode-select --install  (or: brew install git)

    claude — required for agent spawning
      Install: https://docs.anthropic.com/en/docs/claude-code
  ```

### Task 1.4: Write tests for prereq checker
- **File:** `backend/internal/adapter/prereq/check_test.go`
- Test `FormatErrors()` output formatting
- Test platform hint selection (mock `runtime.GOOS` via function injection or test both)
- Test that `Check()` returns empty slice when all tools are present (on dev machine)

## Phase 2: Wire Into Init (2 tasks)

### Task 2.1: Call prereq check at top of runInit
- **File:** `backend/internal/adapter/cli/init.go`
- Add `prereq.Check()` call before config resolution
- If errors returned, print formatted message and return error
- This replaces the later `compose.Detect()` failure for docker compose (but keep `compose.Detect()` for the runner instance)

### Task 2.2: Verify init still works end-to-end
- Run `crelay init` on a machine with all prerequisites — should proceed normally
- Verify no extra output when all tools are present

## Phase 3: Tests and Verification (2 tasks)

### Task 3.1: Add smoke test for prereq check
- **File:** `backend/internal/adapter/cli/init_test.go` or `prereq/check_test.go`
- Test that `Check()` on a dev machine returns no errors (all tools present)
- Test formatting with synthetic errors

### Task 3.2: Run full test suite
- `make test` — all pass
- `make test-smoke` — smoke tests pass
