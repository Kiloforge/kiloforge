# Implementation Plan: Project Add/Remove REST API

**Track ID:** project-manage-api_20260309084650Z

## Phase 1: OpenAPI Schema Update

- [ ] Task 1.1: Add `POST /-/api/projects` to openapi.yaml — request body: `{remote_url: string, name?: string}`, response: `Project` (201) or error (400/409)
- [ ] Task 1.2: Add `DELETE /-/api/projects/{slug}` to openapi.yaml — query param: `cleanup` (boolean, default false), response: 204 or 404
- [ ] Task 1.3: Add request/response schemas: `AddProjectRequest`, error schemas
- [ ] Task 1.4: Regenerate API code: `make gen-api`
- [ ] Task 1.5: Verify: `make verify-codegen` passes

## Phase 2: ProjectStore Remove Method

- [ ] Task 2.1: Add `Remove(slug string) error` to `ProjectStore` port interface
- [ ] Task 2.2: Implement `Remove()` in jsonfile project store — delete from map, persist
- [ ] Task 2.3: Tests for Remove — verify removal, verify 404 on missing slug

## Phase 3: Gitea Client Extensions

- [ ] Task 3.1: Add `DeleteRepo(ctx, repoName)` to Gitea client — `DELETE /api/v1/repos/{owner}/{repo}`
- [ ] Task 3.2: Add `DeleteWebhook(ctx, repoName, hookID)` or `DeleteAllWebhooks(ctx, repoName)` — list + delete hooks
- [ ] Task 3.3: Tests for delete operations

## Phase 4: Add Project Handler

- [ ] Task 4.1: Extract add logic from `cli/add.go` into a reusable service (or call from handler directly)
- [ ] Task 4.2: Implement `AddProject()` handler — validate URL, clone, Gitea setup, register
- [ ] Task 4.3: Return proper HTTP status codes: 201 (created), 400 (bad URL), 409 (duplicate)
- [ ] Task 4.4: Tests for add handler — success, duplicate, invalid URL

## Phase 5: Remove Project Handler

- [ ] Task 5.1: Implement `RemoveProject()` handler — halt agents, deregister from store
- [ ] Task 5.2: If `cleanup=true`: delete Gitea repo, remove filesystem directories
- [ ] Task 5.3: Return 204 on success, 404 on unknown slug
- [ ] Task 5.4: Tests for remove handler — basic remove, remove with cleanup, missing slug

## Phase 6: Final Verification

- [ ] Task 6.1: Run `make build` — compiles cleanly
- [ ] Task 6.2: Run `make test` — all tests pass
- [ ] Task 6.3: Run `make lint` — no lint errors
- [ ] Task 6.4: Verify `make verify-codegen` — generated code matches spec
