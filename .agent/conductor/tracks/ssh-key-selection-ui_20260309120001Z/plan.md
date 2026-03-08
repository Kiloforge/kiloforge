# Implementation Plan: SSH Key Selection in Project Add UI

**Track ID:** ssh-key-selection-ui_20260309120001Z

## Phase 1: Backend API — SSH Key Listing

- [x] Task 1.1: Add `GET /-/api/ssh-keys` endpoint to `openapi.yaml` with `SSHKeyInfo` schema
- [x] Task 1.2: Add `ssh_key` optional field to `AddProjectRequest` schema in `openapi.yaml`
- [x] Task 1.3: Regenerate server code: `oapi-codegen` → `backend/internal/adapter/rest/gen/`
- [x] Task 1.4: Implement `ListSSHKeys` handler in `api_handler.go` — call `auth.DiscoverSSHKeys()`

## Phase 2: Backend — SSH Key in Project Add

- [x] Task 2.1: Update `ProjectService.AddProject()` to accept optional SSH key path
- [x] Task 2.2: When SSH key provided, set `GIT_SSH_COMMAND="ssh -i <key> -o IdentitiesOnly=yes"` for clone operation
- [x] Task 2.3: Update `ProjectManager` interface and `APIHandler.AddProject()` to pass SSH key from request

## Phase 3: Frontend — SSH Key Selector

- [x] Task 3.1: Add `SSHKeyInfo` type and update `AddProjectRequest` in `frontend/src/types/api.ts`
- [x] Task 3.2: Add `fetchSSHKeys()` to `useProjects.ts` hook
- [x] Task 3.3: Update `AddProjectForm.tsx` — detect SSH URL pattern, fetch and display SSH key dropdown
- [x] Task 3.4: Auto-select when single key, hide dropdown for HTTPS URLs
- [x] Task 3.5: Pass selected `ssh_key` in add project request

## Phase 4: Tests & Verification

- [x] Task 4.1: Test `ListSSHKeys` handler returns discovered keys
- [x] Task 4.2: Test `AddProject` with SSH key sets correct GIT_SSH_COMMAND
- [x] Task 4.3: Verify `go test ./...` passes
- [x] Task 4.4: Verify `make build` succeeds
