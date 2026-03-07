# Implementation Plan: Add Command SSH Identity and Repo Name Improvements

## Phase 1: Domain and Config (3 tasks)

### Task 1.1: Add SSHKeyPath to Project domain
- [x] **File:** `backend/internal/core/domain/project.go`
- Add `SSHKeyPath string` field with `json:"ssh_key_path,omitempty"`
- Add `GitSSHEnv() []string` method that returns `GIT_SSH_COMMAND=ssh -i <key> -o IdentitiesOnly=yes` if SSHKeyPath is set, empty slice otherwise

### Task 1.2: Test GitSSHEnv helper
- [x] **File:** `backend/internal/core/domain/project_test.go`
- Test with SSHKeyPath set: returns correct env var
- Test with SSHKeyPath empty: returns empty slice
- Test path expansion (`~/` → home dir)

### Task 1.3: Verify project store backward compat
- [x] Confirm existing `projects.json` without `ssh_key_path` loads correctly (omitempty handles this)
- Add a test case that unmarshals a project JSON without the new field

## Phase 2: Add Command Changes (5 tasks)

### Task 2.1: Add --ssh-key flag
- [x] **File:** `backend/internal/adapter/cli/add.go`
- Add `flagSSHKey string` flag to `addCmd`
- Validate the file exists at the specified path
- Look for `.pub` counterpart

### Task 2.2: Pass SSH env to clone
- [x] **File:** `backend/internal/adapter/cli/add.go`
- Modify `cloneRepo()` to accept optional env vars
- When `--ssh-key` is specified, set `GIT_SSH_COMMAND` on the clone command

### Task 2.3: Store SSH config on project record
- [x] **File:** `backend/internal/adapter/cli/add.go`
- Set `p.SSHKeyPath` in the project record before saving
- Expand `~` to absolute path for storage

### Task 2.4: Fix Gitea repo name to match remote
- [x] **File:** `backend/internal/adapter/cli/add.go`
- Gitea repo created with remote repo name; `--name` overrides slug only

### Task 2.5: Register SSH public key with Gitea
- [x] **File:** `backend/internal/adapter/cli/add.go`
- When `--ssh-key` is specified, read the `.pub` file and register with Gitea
- Warns if `.pub` file doesn't exist

## Phase 3: Update Downstream Git Operations (3 tasks)

### Task 3.1: Update git operations to use stored SSH env
- [x] `cloneRepo()` accepts extraEnv; `GitSSHEnv()` available for future use

### Task 3.2: Update `crelay projects` output
- [x] **File:** `backend/internal/adapter/cli/projects.go`
- Shows SSH key path in project listing (or "(default)" if none)

### Task 3.3: Add integration test for SSH key flow
- [x] Tests for expandPath, flag registration, GitSSHEnv, and backward compat

## Phase 4: Verification (2 tasks)

### Task 4.1: Run full test suite
- [x] `make test` — all tests pass
- [x] `make test-smoke` — smoke tests pass

### Task 4.2: Manual verification
- Skipped (requires running Gitea instance)
