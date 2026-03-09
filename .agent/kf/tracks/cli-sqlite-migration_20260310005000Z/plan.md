# Implementation Plan: Migrate CLI Commands from JSON Files to SQLite

**Track ID:** cli-sqlite-migration_20260310005000Z

## Phase 1: Shared DB Helper

- [x] Task 1.1: Add `openDB(cfg)` helper to CLI package that calls `sqlite.Open(cfg.DataDir)`
- [x] Task 1.2: Verify `sqlite.Open` works correctly when called from CLI commands (not just daemon)

## Phase 2: Migrate CLI Commands

- [x] Task 2.1: Migrate `add.go` — replace `jsonfile.LoadProjectStore()` and `jsonfile.EnsureProjectDir()` with SQLite
- [x] Task 2.2: Migrate `agents.go` — replace `jsonfile.LoadAgentStore()` with SQLite
- [x] Task 2.3: Migrate `attach.go` — replace `jsonfile.LoadAgentStore()` with SQLite
- [x] Task 2.4: Migrate `cost.go` — replace `jsonfile.LoadAgentStore()` and change `*jsonfile.AgentStore` to `port.AgentStore`
- [x] Task 2.5: Migrate `dashboard.go` — replace both `jsonfile.LoadAgentStore()` and `jsonfile.LoadProjectStore()` with SQLite
- [x] Task 2.6: Migrate `escalated.go` — replace `jsonfile.LoadProjectStore()` and `jsonfile.LoadPRTracking()` with SQLite
- [x] Task 2.7: Migrate `projects.go` — replace `jsonfile.LoadProjectStore()` with SQLite
- [x] Task 2.8: Migrate `push.go` — replace `jsonfile.LoadProjectStore()` with SQLite
- [x] Task 2.9: Migrate `status.go` — replace `jsonfile.LoadAgentStore()` and change `*jsonfile.AgentStore` to `port.AgentStore`
- [x] Task 2.10: Migrate `stop.go` — replace `jsonfile.LoadAgentStore()` with SQLite
- [x] Task 2.11: Migrate `sync.go` — replace `jsonfile.LoadProjectStore()` and `jsonfile.NewBoardStore()` with SQLite

## Phase 3: Delete jsonfile Package

- [x] Task 3.1: Delete `backend/internal/adapter/persistence/jsonfile/` directory entirely
- [x] Task 3.2: Grep for any remaining `jsonfile` imports — confirm zero hits
- [x] Task 3.3: Remove `state.json` and `projects.json` from any documentation or gitignore references if applicable

## Phase 4: Verification

- [x] Task 4.1: `make build` succeeds
- [x] Task 4.2: `make test` passes
- [ ] Task 4.3: Manual smoke test — `kf agents`, `kf projects`, `kf status`, `kf cost` read daemon-written data correctly
