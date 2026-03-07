# Implementation Plan: Refactor Config to Port/Adapter Pattern with Layered Resolution

**Track ID:** refactor-config-port-adapter_20260307121000Z

## Phase 1: Define Port and Core Types

### Task 1.1: Define ConfigProvider interface and expand Config struct [x]
### Task 1.2: Implement defaults adapter [x]

### Verification 1
- [x] ConfigProvider interface defined
- [x] Config struct includes all previously-constant fields
- [x] Defaults adapter returns complete config with expected values
- [x] Tests pass

## Phase 2: Implement Adapters

### Task 2.1: Refactor JSON file adapter [x]
### Task 2.2: Implement env var adapter [x]
### Task 2.3: Implement flags adapter [x]

### Verification 2
- [x] JSON adapter reads/writes correctly
- [x] Env adapter reads all CRELAY_* vars
- [x] Flags adapter only returns explicitly set values
- [x] All adapter tests pass

## Phase 3: Implement Merger and Resolution

### Task 3.1: Implement config merger [x]
### Task 3.2: Create top-level Resolve function [x]

### Verification 3
- [x] Merger correctly applies priority ordering
- [x] Resolve() returns fully resolved config
- [x] flags > env > json > defaults ordering confirmed in tests
- [x] Edge cases: missing JSON file, no env vars set, no flags — still works

## Phase 4: Migrate Call Sites

### Task 4.1: Update init.go [x]
### Task 4.2: Update status.go, destroy.go, and compose-dependent commands [x]
### Task 4.3: Update gitea/manager.go, agent/spawner.go, relay/server.go [x]
### Task 4.4: Update remaining CLI commands (agents, logs, attach, stop) [x]
### Task 4.5: Update existing tests and clean up [x]

### Verification 4
- [x] All call sites use `config.Resolve()` or adapter chain
- [x] No references to removed constants remain
- [x] `go build ./...` succeeds
- [x] `go test ./...` passes
- [x] golangci-lint — no new issues (pre-existing issues unrelated to this refactor)
