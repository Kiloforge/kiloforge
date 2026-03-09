# Implementation Plan: Branch Diff View API (Backend)

**Track ID:** branch-diff-view-be_20260310050000Z

## Phase 1: Diff Parser and Git Adapter (Foundation)

### Task 1.1: Define diff domain types
- Create `internal/core/domain/diff.go` with types: `DiffResult`, `FileDiff`, `Hunk`, `DiffLine`, `DiffStats`, `LineSide` (add/delete/context)
- Include `BranchInfo` type for branch listing response

### Task 1.2: Write diff parser with tests first (TDD)
- Create `internal/adapter/git/diffparse.go` — pure function `ParseUnifiedDiff(raw string) ([]FileDiff, error)`
- Create `internal/adapter/git/diffparse_test.go` with table-driven tests:
  - Modified file with multiple hunks
  - New file (--- /dev/null)
  - Deleted file (+++ /dev/null)
  - Renamed file (with similarity index)
  - Binary file detection
  - Empty diff (no changes)
  - File with no newline at EOF marker
  - Multiple files in one diff
- Parse `@@ -old,count +new,count @@` headers for line numbers
- Classify lines: `+` → add, `-` → delete, ` ` → context
- Track per-file insertion/deletion counts

### Task 1.3: Implement git diff adapter
- Create `internal/adapter/git/diff.go` with:
  - `Diff(projectDir, branch string) (*domain.DiffResult, error)` — runs `git diff main...{branch} -U3 --no-color` and parses output
  - `DiffStats(projectDir, branch string) (*domain.DiffStats, error)` — runs `git diff main...{branch} --stat --numstat` for summary
- Use `exec.CommandContext` with timeout (30s) to prevent hanging on large diffs
- Handle branch-not-found error (git exit code)

### Task 1.4: Verify Phase 1
- Run diff parser tests — all passing
- Verify diff adapter against a real git repo with known changes

## Phase 2: API Endpoints (Core)

### Task 2.1: Update OpenAPI spec
- Add `GET /api/projects/{slug}/diff` with query param `branch` (required)
  - 200: DiffResponse schema (stats + files array)
  - 404: branch not found
  - 400: missing branch param
- Add `GET /api/projects/{slug}/branches`
  - 200: array of BranchInfo (branch name, agent_id, track_id, status)
- Add all response schemas: DiffResponse, FileDiff, Hunk, DiffLine, DiffStats, BranchInfo

### Task 2.2: Regenerate OpenAPI server code
- Run `make generate` (or oapi-codegen) to regenerate handler interface and types
- Verify new methods appear in `ServerInterface`

### Task 2.3: Implement diff endpoint handler
- In `api_handler.go`, implement `GetProjectDiff(w, r, slug, params)`:
  - Extract `branch` query param
  - Look up project by slug
  - Call diff adapter with project dir and branch
  - Return structured JSON response
  - Handle errors: project not found, branch not found, diff too large

### Task 2.4: Implement branches endpoint handler
- In `api_handler.go`, implement `GetProjectBranches(w, r, slug)`:
  - Look up project by slug
  - Query pool for all worktrees associated with this project
  - Return branch name, agent ID, track ID, worktree status for each

### Task 2.5: Write endpoint tests
- Test diff endpoint with mock git adapter
- Test branches endpoint with mock pool
- Test error cases: missing branch param, project not found, branch not found

### Task 2.6: Verify Phase 2
- Run all tests — passing
- Manual curl test of both endpoints against a running instance

## Phase 3: Integration and Polish

### Task 3.1: Add diff port interface
- Create `DiffProvider` interface in `internal/core/port/` if needed for testability
- Wire into REST handler via dependency injection (consistent with existing patterns)

### Task 3.2: Handle large diffs
- Add a `max_files` query param (default 100) to limit response size
- If diff exceeds limit, return truncated response with `truncated: true` flag and total file count
- Binary files: include in file list with `is_binary: true` but empty hunks

### Task 3.3: Final integration test
- End-to-end test: create a branch with known changes, call diff endpoint, verify structured response matches expected output

### Task 3.4: Verify Phase 3
- All tests passing
- `make test` clean
- `make build` clean

---

**Total: 12 tasks across 3 phases**
