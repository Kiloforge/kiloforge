# Implementation Plan: Rename Relay Server to Orchestrator

**Track ID:** rename-relay-orchestrator_20260309075537Z

## Phase 1: Config Layer Rename

- [x] Task 1.1: Rename `RelayPort` → `OrchestratorPort` in `config.go` struct + JSON tag (`orchestrator_port`)
- [x] Task 1.2: Rename `WithRelayPort()` → `WithOrchestratorPort()` in `flags_adapter.go`
- [x] Task 1.3: Update default value assignment in `defaults.go`
- [x] Task 1.4: Update merge logic in `merger.go`
- [x] Task 1.5: Update env adapter if `KF_RELAY_PORT` env var exists → `KF_ORCH_PORT`
- [x] Task 1.6: Update all config tests (`defaults_test.go`, `resolve_test.go`, `env_adapter_test.go`)
- [x] Task 1.7: Verify: `make test` passes for config package

## Phase 2: CLI Messages and Logging

- [x] Task 2.1: Update `serve.go` — log prefix `[relay]` → `[orchestrator]`, log file `relay.log` → `orchestrator.log`, startup message
- [x] Task 2.2: Update `daemon.go` — log file path, stop daemon comment
- [x] Task 2.3: Update `init.go` — all "relay" user-facing messages → "orchestrator"
- [x] Task 2.4: Update `up.go` — command description, all "relay" messages → "orchestrator"
- [x] Task 2.5: Update `down.go` — command description, all "relay" messages → "orchestrator"
- [x] Task 2.6: Update `destroy.go` — "relay daemon" messages → "orchestrator"
- [x] Task 2.7: Update `status.go` — command description, variable names (`relayRunning` → `orchRunning`)
- [x] Task 2.8: Update `dashboard.go` — command description referencing relay
- [x] Task 2.9: Update `add.go` — webhook relay message
- [x] Task 2.10: Verify: `make build` succeeds

## Phase 3: REST Server and Gitea Client

- [x] Task 3.1: Update `server.go` — comments, log prefix `[relay]` → `[orchestrator]`
- [x] Task 3.2: Update `client.go` — rename `relayPort` parameter to `orchPort` in `CreateWebhook`
- [x] Task 3.3: Update all callers of `CreateWebhook` that pass `RelayPort`
- [x] Task 3.4: Verify: `make test` passes for rest and gitea packages

## Phase 4: Compose Template

- [x] Task 4.1: Rename `RelayPort` → `OrchestratorPort` in compose template struct
- [x] Task 4.2: Update template string using `.RelayPort` → `.OrchestratorPort`
- [x] Task 4.3: Update `template_test.go` — struct field names and test descriptions
- [x] Task 4.4: Verify: `make test` passes for compose package

## Phase 5: API Specs and Documentation

- [x] Task 5.1: Update AsyncAPI spec description — "relay server" → "orchestrator"
- [x] Task 5.2: Update `README.md` — all "relay" references → "orchestrator"
- [x] Task 5.3: Update `backend/docs/architecture.md` if it references "relay"
- [x] Task 5.4: Update `backend/docs/commands.md` if it references "relay"
- [x] Task 5.5: Update `.agent/conductor/product.md` — product description

## Phase 6: Environment Variable Rename

- [x] Task 6.1: Rename `KF_RELAY_URL` → `KF_ORCH_URL` in env adapter
- [x] Task 6.2: Update env adapter tests
- [x] Task 6.3: Update skill files in `skills/` (if kf-skills-source track has completed) — `KF_RELAY_URL` → `KF_ORCH_URL`
- [x] Task 6.4: Search for any remaining `RELAY` references in env vars and update

## Phase 7: Final Verification

- [x] Task 7.1: Run `make build` — compiles cleanly
- [x] Task 7.2: Run `make test` — all tests pass
- [x] Task 7.3: Grep for remaining "relay" references — verify only historical research docs and track IDs remain
- [x] Task 7.4: Verify CLI messaging via `kf up --help`
