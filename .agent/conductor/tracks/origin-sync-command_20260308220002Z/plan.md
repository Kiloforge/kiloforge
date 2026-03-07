# Implementation Plan: Origin Sync Command

## Phase 1: Command Scaffold (3 tasks)

### Task 1.1: Create push command
- **File:** `backend/internal/adapter/cli/push.go`
- Add `pushCmd` with `Use: "push [slug]"`, `--branch` flag (default: main), `--all` flag
- Register in `root.go`
- Load config and project registry

### Task 1.2: Implement single-project push
- **File:** `backend/internal/adapter/cli/push.go`
- Look up project by slug from registry
- Validate `OriginRemote` is set (error if not)
- Build `git -C <projectDir> push origin <branch>` command
- Set `GIT_SSH_COMMAND` env from `project.GitSSHEnv()` if SSH key is configured
- Execute and report stdout/stderr

### Task 1.3: Implement --all flag
- **File:** `backend/internal/adapter/cli/push.go`
- Iterate `reg.List()`, push each project's main branch
- Report per-project success/failure
- Continue on individual failures (don't abort all)

## Phase 2: Status Reporting (2 tasks)

### Task 2.1: Add ahead/behind check before push
- **File:** `backend/internal/adapter/cli/push.go`
- Run `git -C <projectDir> fetch origin <branch>` first (using SSH env)
- Run `git -C <projectDir> rev-list --left-right --count origin/<branch>...HEAD`
- Report "N commits ahead, M commits behind" before pushing
- Warn if behind (push may fail)

### Task 2.2: Handle push errors gracefully
- Non-fast-forward: report "origin has diverged, manual resolution needed"
- Auth failure: report "SSH key may be incorrect, check --ssh-key"
- Remote not found: report "origin remote not configured"

## Phase 3: Tests (3 tasks)

### Task 3.1: Unit test push command logic
- **File:** `backend/internal/adapter/cli/push_test.go`
- Test slug lookup from registry
- Test SSH env is set when SSHKeyPath present
- Test --all iterates all projects

### Task 3.2: Test git command construction
- Verify correct args: `git -C <dir> push origin main`
- Verify GIT_SSH_COMMAND env var is set correctly
- Verify custom branch: `git -C <dir> push origin feature-x`

### Task 3.3: Run full test suite
- `make test` — all pass
- `make test-smoke` — smoke tests pass
