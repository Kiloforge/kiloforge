# Implementation Plan: Origin Sync Command

## Phase 1: Command Scaffold (3 tasks)

### Task 1.1: Create push command
- [x] **File:** `backend/internal/adapter/cli/push.go`
- Add `pushCmd` with `Use: "push [slug]"`, `--branch` flag (default: main), `--all` flag
- Register in `root.go`
- Load config and project registry

### Task 1.2: Implement single-project push
- [x] **File:** `backend/internal/adapter/cli/push.go`
- Look up project by slug from registry
- Validate `OriginRemote` is set (error if not)
- Build `git -C <projectDir> push origin <branch>` command
- Set `GIT_SSH_COMMAND` env from `project.GitSSHEnv()` if SSH key is configured
- Execute and report stdout/stderr

### Task 1.3: Implement --all flag
- [x] **File:** `backend/internal/adapter/cli/push.go`
- Iterate `reg.List()`, push each project's main branch
- Report per-project success/failure
- Continue on individual failures (don't abort all)

## Phase 2: Status Reporting (2 tasks)

### Task 2.1: Add ahead/behind check before push
- [x] **File:** `backend/internal/adapter/cli/push.go`
- Fetch origin, rev-list ahead/behind, warn if behind

### Task 2.2: Handle push errors gracefully
- [x] Non-fast-forward detection, no-origin-remote error, per-project failure reporting

## Phase 3: Tests (3 tasks)

### Task 3.1: Unit test push command logic
- [x] **File:** `backend/internal/adapter/cli/push_test.go`
- Tests for flag registration, slug/all validation, empty projects

### Task 3.2: Test git command construction
- [x] Tests for gitCmd with/without SSH env, custom branch args

### Task 3.3: Run full test suite
- [x] `make test` — all pass
