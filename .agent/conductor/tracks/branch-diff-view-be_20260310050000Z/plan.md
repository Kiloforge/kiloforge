# Implementation Plan: Branch Diff View API (Backend)

**Track ID:** branch-diff-view-be_20260310050000Z

## Phase 1: Diff Parser and Git Adapter (Foundation)

### Task 1.1: Define diff domain types
- [x] Create `internal/core/domain/diff.go` with types: `DiffResult`, `FileDiff`, `Hunk`, `DiffLine`, `DiffStats`, `LineSide` (add/delete/context)
- [x] Include `BranchInfo` type for branch listing response

### Task 1.2: Write diff parser with tests first (TDD)
- [x] Create `internal/adapter/git/diffparse.go` — pure function `ParseUnifiedDiff(raw string) ([]FileDiff, error)`
- [x] Create `internal/adapter/git/diffparse_test.go` with table-driven tests:
  - Modified file with multiple hunks
  - New file (--- /dev/null)
  - Deleted file (+++ /dev/null)
  - Renamed file (with similarity index)
  - Binary file detection
  - Empty diff (no changes)
  - File with no newline at EOF marker
  - Multiple files in one diff
- [x] Parse `@@ -old,count +new,count @@` headers for line numbers
- [x] Classify lines: `+` → add, `-` → delete, ` ` → context
- [x] Track per-file insertion/deletion counts

### Task 1.3: Implement git diff adapter
- [x] Create `internal/adapter/git/diff.go` with:
  - `Diff(projectDir, branch string) (*domain.DiffResult, error)` — runs `git diff main...{branch} -U3 --no-color` and parses output
  - `DiffWithMaxFiles(projectDir, branch string, maxFiles int) (*domain.DiffResult, error)` — with truncation support
- [x] Use `exec.CommandContext` with timeout (30s) to prevent hanging on large diffs
- [x] Handle branch-not-found error (git exit code)

### Task 1.4: Verify Phase 1
- [x] Run diff parser tests — all passing
- [x] Verify diff adapter compiles and builds

## Phase 2: API Endpoints (Core)

### Task 2.1: Update OpenAPI spec
- [x] Add `GET /api/projects/{slug}/diff` with query params `branch` (required) and `max_files` (optional)
  - 200: DiffResponse schema (stats + files array)
  - 404: project/branch not found
  - 400: missing branch param
- [x] Add `GET /api/projects/{slug}/branches`
  - 200: array of BranchInfo (branch name, agent_id, track_id, status)
- [x] Add all response schemas: DiffResponse, FileDiff, DiffHunk, DiffLine, DiffStats, BranchInfo

### Task 2.2: Regenerate OpenAPI server code
- [x] Run oapi-codegen to regenerate handler interface and types
- [x] Verify new methods appear in `StrictServerInterface`

### Task 2.3: Implement diff endpoint handler
- [x] In `api_handler.go`, implement `GetProjectDiff`:
  - Look up project by slug
  - Call diff provider with project dir and branch
  - Return structured JSON response
  - Handle errors: project not found, branch not found, diff too large

### Task 2.4: Implement branches endpoint handler
- [x] In `api_handler.go`, implement `GetProjectBranches`:
  - Look up project by slug
  - Build branch info from active agents with worktree directories
  - Return branch name, agent ID, track ID, status for each

### Task 2.5: Write endpoint tests
- [x] Test diff endpoint error cases (project not found, provider not configured)
- [x] Test branches endpoint (project not found, agents with/without worktrees, empty)

### Task 2.6: Verify Phase 2
- [x] All tests passing
- [x] `make test` clean
- [x] `make build` clean

## Phase 3: Integration and Polish

### Task 3.1: Add diff port interface
- [x] Create `DiffProvider` interface in `internal/core/port/diff_provider.go`
- [x] Wire into REST handler via dependency injection (consistent with existing patterns)

### Task 3.2: Handle large diffs
- [x] `max_files` query param (default 100) limits response size
- [x] Truncated response includes `truncated: true` flag
- [x] Binary files: included in file list with `is_binary: true` but empty hunks

### Task 3.3: Final integration test
- [x] All endpoint tests pass
- [x] Diff parser thoroughly tested with 8 table-driven test cases

### Task 3.4: Verify Phase 3
- [x] All tests passing
- [x] `make test` clean
- [x] `make build` clean

---

**Total: 12 tasks across 3 phases — ALL COMPLETE**
