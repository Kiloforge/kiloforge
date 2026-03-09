# Implementation Plan: Origin Push/Pull REST API with Remote Branch Targeting

**Track ID:** origin-sync-api_20260309143000Z

## Phase 1: Git Sync Adapter

- [x] Task 1.1: Create `backend/internal/adapter/git/sync.go` — `GitSync` struct with `FetchOrigin()`, `PushToRemote()`, `PullFromRemote()`, `SyncStatus()` methods
- [x] Task 1.2: Implement SSH key env setup (reuse pattern from `cli/push.go`)
- [x] Task 1.3: Implement `PushToRemote(projectDir, localBranch, remoteBranch, sshKeyPath)` — runs `git push origin localBranch:refs/heads/remoteBranch`
- [x] Task 1.4: Implement `PullFromRemote(projectDir, remoteBranch, sshKeyPath)` — fetch + fast-forward merge
- [x] Task 1.5: Implement `SyncStatus(projectDir)` — returns ahead/behind counts via `git rev-list`
- [x] Task 1.6: Add tests for git sync operations (using test git repos)

## Phase 2: OpenAPI Spec and Endpoints

- [x] Task 2.1: Add `POST /api/projects/{slug}/push`, `POST /api/projects/{slug}/pull`, `GET /api/projects/{slug}/sync-status` to `openapi.yaml` with request/response schemas
- [x] Task 2.2: Regenerate server code
- [x] Task 2.3: Implement `PushProject()` handler — validate slug, get project, call `GitSync.PushToRemote()`
- [x] Task 2.4: Implement `PullProject()` handler — validate slug, get project, call `GitSync.PullFromRemote()`
- [x] Task 2.5: Implement `GetSyncStatus()` handler — call `GitSync.SyncStatus()`, return structured response
- [x] Task 2.6: Wire `GitSync` into REST server

## Phase 3: Verification

- [x] Task 3.1: Verify `go test ./...` passes
- [x] Task 3.2: Verify `make build` succeeds
- [x] Task 3.3: Verify push endpoint works with SSH key auth
