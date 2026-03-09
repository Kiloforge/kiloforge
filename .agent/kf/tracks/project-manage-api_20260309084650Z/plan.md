# Implementation Plan: Project Add/Remove REST API

**Track ID:** project-manage-api_20260309084650Z

## Phase 1: OpenAPI Schema Update

- [x] Task 1.1: Add `POST /-/api/projects` to openapi.yaml — request body: `{remote_url: string, name?: string}`, response: `Project` (201) or error (400/409)
- [x] Task 1.2: Add `DELETE /-/api/projects/{slug}` to openapi.yaml — query param: `cleanup` (boolean, default false), response: 204 or 404
- [x] Task 1.3: Add request/response schemas: `AddProjectRequest`, error schemas
- [x] Task 1.4: Regenerate API code: `make gen-api`
- [x] Task 1.5: Verify: `make verify-codegen` passes

## Phase 2: ProjectStore Remove Method

- [x] Task 2.1: Add `Remove(slug string) error` to `ProjectStore` port interface
- [x] Task 2.2: Implement `Remove()` in jsonfile project store — delete from map, persist
- [x] Task 2.3: Tests for Remove — verify removal, verify 404 on missing slug

## Phase 3: Gitea Client Extensions

- [x] Task 3.1: Add `DeleteRepo(ctx, repoName)` to Gitea client — `DELETE /api/v1/repos/{owner}/{repo}`
- [x] Task 3.2: Add `DeleteAllWebhooks(ctx, repoName)` — list + delete hooks
- [x] Task 3.3: Tests for delete operations

## Phase 4: Add Project Handler

- [x] Task 4.1: Extract add logic from `cli/add.go` into a reusable service (or call from handler directly)
- [x] Task 4.2: Implement `AddProject()` handler — validate URL, clone, Gitea setup, register
- [x] Task 4.3: Return proper HTTP status codes: 201 (created), 400 (bad URL), 409 (duplicate)
- [x] Task 4.4: Tests for add handler — success, duplicate, invalid URL

## Phase 5: Remove Project Handler

- [x] Task 5.1: Implement `RemoveProject()` handler — halt agents, deregister from store
- [x] Task 5.2: If `cleanup=true`: delete Gitea repo, remove filesystem directories
- [x] Task 5.3: Return 204 on success, 404 on unknown slug
- [x] Task 5.4: Tests for remove handler — basic remove, remove with cleanup, missing slug

## Phase 6: Final Verification

- [x] Task 6.1: Run `make build` — compiles cleanly
- [x] Task 6.2: Run `make test` — all tests pass
- [x] Task 6.3: Run `make lint` — skipped (not in workflow.md verification commands)
- [x] Task 6.4: Verify `make verify-codegen` — generated code matches spec
