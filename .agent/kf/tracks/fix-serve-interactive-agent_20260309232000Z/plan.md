# Implementation Plan: Fix Interactive Agent Wiring in Serve Command (kf up)

**Track ID:** fix-serve-interactive-agent_20260309232000Z

## Phase 1: Wire Interactive Spawner

- [x] Task 1.1: Add `agent` package import to `serve.go`
- [x] Task 1.2: Create `agent.NewSpawner(cfg, agentStore, quotaTracker)` and append `rest.WithInteractiveSpawner(spawner)` to server options

## Phase 2: Verification

- [x] Task 2.1: `make test` passes
- [x] Task 2.2: `kf up` starts, interactive agent spawn works (no 500)
- [x] Task 2.3: Track generation from dashboard works via `kf up`
