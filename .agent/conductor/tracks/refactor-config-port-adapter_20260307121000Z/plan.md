# Implementation Plan: Refactor Config to Port/Adapter Pattern with Layered Resolution

**Track ID:** refactor-config-port-adapter_20260307121000Z

## Phase 1: Define Port and Core Types

### Task 1.1: Define ConfigProvider interface and expand Config struct
- Define `ConfigProvider` interface in `internal/config/port.go`
- Expand `Config` struct to include previously hardcoded constants as fields: `ContainerName`, `GiteaImage`, `GiteaAdminUser`, `GiteaAdminPass`, `GiteaAdminEmail`
- Remove the package-level constants for these values (they move to defaults adapter)
- Keep `ConfigFileName` as a constant (it's infrastructure, not user-configurable)
- Keep `GiteaURL()` method on Config
- Tests: verify Config struct serialization with new fields

### Task 1.2: Implement defaults adapter
- Create `internal/config/defaults.go`
- Implements `ConfigProvider` — returns a Config with all default values:
  - `GiteaPort: 3000`
  - `DataDir: ~/.crelay`
  - `ContainerName: "conductor-gitea"`
  - `GiteaImage: "gitea/gitea:latest"`
  - `GiteaAdminUser: "conductor"`
  - `GiteaAdminPass: "conductor123"`
  - `GiteaAdminEmail: "conductor@local.dev"`
- Tests: verify all defaults are returned, verify home dir resolution for DataDir

### Verification 1
- [ ] ConfigProvider interface defined
- [ ] Config struct includes all previously-constant fields
- [ ] Defaults adapter returns complete config with expected values
- [ ] Tests pass

## Phase 2: Implement Adapters

### Task 2.1: Refactor JSON file adapter
- Create `internal/config/json_adapter.go`
- Implements `ConfigProvider` — reads from JSON file at `{dataDir}/config.json`
- Also implements `Save(*Config) error` for write-back (not part of the read port)
- Constructor: `NewJSONAdapter(dataDir string) *JSONAdapter`
- Returns only fields present in the file (zero-value for unset fields)
- Move existing `Load`, `LoadFrom`, `Save` logic here
- Tests: save and load round-trip, missing file returns zero Config (not error), partial file

### Task 2.2: Implement env var adapter
- Create `internal/config/env_adapter.go`
- Implements `ConfigProvider` — reads `CRELAY_*` env vars
- Maps: `CRELAY_GITEA_PORT` → GiteaPort, `CRELAY_DATA_DIR` → DataDir, etc. (full mapping per spec)
- Parses int fields from string env vars
- Returns zero-value for unset env vars (so merger skips them)
- Tests: set env vars, verify Load returns expected values; unset vars return zero

### Task 2.3: Implement flags adapter
- Create `internal/config/flags_adapter.go`
- Implements `ConfigProvider`
- Constructor: `NewFlagsAdapter(opts ...FlagOption) *FlagsAdapter`
- `FlagOption` pattern: `WithGiteaPort(int)`, `WithDataDir(string)`, etc.
- Only sets fields for flags that were explicitly provided (not default flag values)
- Tests: verify only explicitly set fields are returned, others are zero

### Verification 2
- [ ] JSON adapter reads/writes correctly
- [ ] Env adapter reads all CRELAY_* vars
- [ ] Flags adapter only returns explicitly set values
- [ ] All adapter tests pass

## Phase 3: Implement Merger and Resolution

### Task 3.1: Implement config merger
- Create `internal/config/merger.go`
- `Merge(providers ...ConfigProvider) (*Config, error)` — chains providers, applies in order (first = lowest priority, last = highest)
- For each provider, call `Load()` and overlay non-zero fields onto accumulator
- Non-zero detection: `0` for int, `""` for string — these mean "not set"
- If a field genuinely needs to be zero/empty, that's handled by defaults being first in chain
- Tests: verify priority ordering, verify partial overlays, verify full chain (defaults + json + env + flags)

### Task 3.2: Create top-level Resolve function
- Add `Resolve(providers ...ConfigProvider) (*Config, error)` as the primary public API
- Default call (no args): chains defaults → JSON → env
- With flags: chains defaults → JSON → env → flags
- Replace old `Load()` / `LoadFrom()` with `Resolve()` (keep `Load` as deprecated wrapper for transition)
- Tests: integration test with all layers

### Verification 3
- [ ] Merger correctly applies priority ordering
- [ ] Resolve() returns fully resolved config
- [ ] flags > env > json > defaults ordering confirmed in tests
- [ ] Edge cases: missing JSON file, no env vars set, no flags — still works

## Phase 4: Migrate Call Sites

### Task 4.1: Update init.go
- Replace `&config.Config{}` struct literal with flags adapter + `config.Resolve()`
- Pass flag values via `NewFlagsAdapter(WithGiteaPort(...), WithDataDir(...))`
- Replace `config.GiteaAdminUser` / `config.GiteaAdminPass` constants with resolved config fields
- Replace `config.LoadFrom()` with `config.Resolve()` for idempotency check
- Keep `cfg.Save()` via JSON adapter for persistence

### Task 4.2: Update status.go, destroy.go, and compose-dependent commands
- Replace `config.Load()` with `config.Resolve()`
- Replace `config.GiteaAdminUser` / `config.GiteaAdminPass` constant references with config fields
- `status.go`: update status output to show resolved config source info if useful
- `destroy.go`: use resolved config for data dir

### Task 4.3: Update gitea/manager.go, agent/spawner.go, relay/server.go
- Replace all `config.GiteaAdminUser`, `config.GiteaAdminPass`, `config.GiteaAdminEmail` constant references with `cfg.GiteaAdminUser`, `cfg.GiteaAdminPass`, `cfg.GiteaAdminEmail` fields
- Replace `config.ContainerName` with `cfg.ContainerName` — pass through via Config
- Replace `config.GiteaImage` with `cfg.GiteaImage` in compose template (if used there)
- `server.go`: update `config.GiteaAdminUser` reference

### Task 4.4: Update remaining CLI commands (agents, logs, attach, stop)
- Replace `config.Load()` with `config.Resolve()` in each
- These commands only use `cfg.DataDir` so the change is minimal

### Task 4.5: Update existing tests and clean up
- Update `config_test.go` to test new adapter/merger architecture
- Remove old `Load()` / `LoadFrom()` if fully replaced, or keep as thin wrappers
- Remove deleted constants from any remaining references
- Verify `go build ./...` and `go test ./...` pass
- Run `golangci-lint` to catch any issues

### Verification 4
- [ ] All call sites use `config.Resolve()` or adapter chain
- [ ] No references to removed constants remain
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `golangci-lint run ./...` clean
