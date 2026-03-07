# Implementation Plan: Add Command SSH Identity and Repo Name Improvements

## Phase 1: Domain and Config (3 tasks)

### Task 1.1: Add SSHKeyPath to Project domain
- **File:** `backend/internal/core/domain/project.go`
- Add `SSHKeyPath string` field with `json:"ssh_key_path,omitempty"`
- Add `GitSSHEnv() []string` method that returns `GIT_SSH_COMMAND=ssh -i <key> -o IdentitiesOnly=yes` if SSHKeyPath is set, empty slice otherwise

### Task 1.2: Test GitSSHEnv helper
- **File:** `backend/internal/core/domain/project_test.go`
- Test with SSHKeyPath set: returns correct env var
- Test with SSHKeyPath empty: returns empty slice
- Test path expansion (`~/` → home dir)

### Task 1.3: Verify project store backward compat
- Confirm existing `projects.json` without `ssh_key_path` loads correctly (omitempty handles this)
- Add a test case that unmarshals a project JSON without the new field

## Phase 2: Add Command Changes (5 tasks)

### Task 2.1: Add --ssh-key flag
- **File:** `backend/internal/adapter/cli/add.go`
- Add `flagSSHKey string` flag to `addCmd`
- Validate the file exists at the specified path
- Look for `.pub` counterpart

### Task 2.2: Pass SSH env to clone
- **File:** `backend/internal/adapter/cli/add.go`
- Modify `cloneRepo()` to accept optional env vars
- When `--ssh-key` is specified, set `GIT_SSH_COMMAND` on the clone command
- Also set it on the `git push` command to Gitea (if needed)

### Task 2.3: Store SSH config on project record
- **File:** `backend/internal/adapter/cli/add.go`
- Set `p.SSHKeyPath` in the project record before saving
- Expand `~` to absolute path for storage

### Task 2.4: Fix Gitea repo name to match remote
- **File:** `backend/internal/adapter/cli/add.go`
- Currently slug = repoName = derived from URL. Ensure Gitea repo is created with the name from the URL
- `--name` overrides the slug (used for local directory and registry key) but the Gitea repo name should still match the remote repo name unless `--name` is given

### Task 2.5: Register SSH public key with Gitea
- **File:** `backend/internal/adapter/cli/add.go`
- When `--ssh-key` is specified, read the `.pub` file and register it with Gitea via `client.AddSSHKey()`
- Skip if `.pub` file doesn't exist (warn the user)

## Phase 3: Update Downstream Git Operations (3 tasks)

### Task 3.1: Update git operations to use stored SSH env
- **Files:** Any code that runs git commands against a project's origin remote
- Use `project.GitSSHEnv()` when setting up `exec.Cmd.Env`
- Currently: `cli/add.go` (push), future: sync command

### Task 3.2: Update `crelay projects` output
- **File:** `backend/internal/adapter/cli/projects.go`
- Show SSH key path in project listing when configured

### Task 3.3: Add integration test for SSH key flow
- Test that `--ssh-key` flag is accepted and stored
- Test that `GitSSHEnv()` produces correct env for stored key
- Test backward compat: project without SSH key works normally

## Phase 4: Verification (2 tasks)

### Task 4.1: Run full test suite
- `make test` — all tests pass
- `make test-smoke` — smoke tests pass

### Task 4.2: Manual verification
- Test `crelay add <remote> --ssh-key ~/.ssh/id_ed25519_goblinlordx` with a real repo
- Verify clone uses correct key
- Verify project record has SSH key stored
