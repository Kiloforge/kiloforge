# Implementation Plan: Fix Interactive Agent Wiring in Serve Command (kf up)

**Track ID:** fix-serve-interactive-agent_20260309232000Z

## Phase 1: Wire Interactive Spawner

- [ ] Task 1.1: Add `agent` package import to `serve.go`
- [ ] Task 1.2: Create `agent.NewSpawner(cfg, agentStore, quotaStore)` and append `rest.WithInteractiveSpawner(spawner)` to server options

## Phase 2: Verification

- [ ] Task 2.1: `make test` passes
- [ ] Task 2.2: `kf up` starts, interactive agent spawn works (no 500)
- [ ] Task 2.3: Track generation from dashboard works via `kf up`
