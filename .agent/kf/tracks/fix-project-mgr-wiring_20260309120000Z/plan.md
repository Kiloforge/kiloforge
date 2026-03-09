# Implementation Plan: Fix Project Manager Wiring in REST Server

**Track ID:** fix-project-mgr-wiring_20260309120000Z

## Phase 1: Wire ProjectService

- [x] Task 1.1: In `server.go` `Run()`, create `ProjectService` using `s.registry`, `s.client`, and config values from `s.cfg`
- [x] Task 1.2: Pass the `ProjectService` as `ProjectMgr` in `APIHandlerOpts`

## Phase 2: Verification

- [x] Task 2.1: Verified API handler wiring compiles correctly
- [x] Task 2.2: Existing API handler tests pass (add/remove project endpoints tested via stubProjectManager)
- [x] Task 2.3: `go test ./...` passes
- [x] Task 2.4: `make build` succeeds
