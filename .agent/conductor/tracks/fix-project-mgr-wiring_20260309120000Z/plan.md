# Implementation Plan: Fix Project Manager Wiring in REST Server

**Track ID:** fix-project-mgr-wiring_20260309120000Z

## Phase 1: Wire ProjectService

- [ ] Task 1.1: In `server.go` `setupRoutes()`, create `ProjectService` using `s.registry`, `s.client`, and config values from `s.cfg`
- [ ] Task 1.2: Pass the `ProjectService` as `ProjectMgr` in `APIHandlerOpts`

## Phase 2: Verification

- [ ] Task 2.1: Verify `POST /-/api/projects` returns 201 (not 500)
- [ ] Task 2.2: Verify `DELETE /-/api/projects/{slug}` works
- [ ] Task 2.3: Verify `go test ./...` passes
- [ ] Task 2.4: Verify `make build` succeeds
