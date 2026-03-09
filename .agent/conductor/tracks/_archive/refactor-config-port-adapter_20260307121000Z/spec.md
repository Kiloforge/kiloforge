# Specification: Refactor Config to Port/Adapter Pattern with Layered Resolution

**Track ID:** refactor-config-port-adapter_20260307121000Z
**Type:** Refactor
**Created:** 2026-03-07T12:10:00Z
**Status:** Draft

## Summary

Refactor `internal/config` from a concrete JSON-only implementation to a port/adapter architecture with layered resolution. Define a `ConfigPort` interface, implement four adapters (defaults, JSON file, env vars, CLI flags), and a merger that resolves values in priority order: flags > env > JSON file > defaults.

## Context

The current `internal/config/config.go` couples config reading, writing, and defaults into a single concrete struct with JSON file persistence. Several values are hardcoded as package-level constants (`ContainerName`, `GiteaImage`, `GiteaAdminUser`, etc.) rather than being configurable. There is no support for environment variable overrides, which is needed for CI, containers, and scripting. The refactor follows hexagonal architecture patterns already established in the project.

## Codebase Analysis

### Current config.go (58 lines)
- **Constants (lines 10-17):** `ContainerName`, `GiteaImage`, `GiteaAdminUser`, `GiteaAdminPass`, `GiteaAdminEmail`, `ConfigFileName` — all hardcoded, not overridable
- **Config struct (lines 19-24):** `GiteaPort`, `DataDir`, `APIToken`, `ComposeFile`
- **Methods:** `GiteaURL()`, `Save()`, `Load()`, `LoadFrom()`
- **No defaults logic** — defaults are scattered across CLI flag definitions in `init.go` (`--gitea-port 3000`, `--data-dir ~/.kiloforge`)

### Call sites (10 files)
- **`config.Load()`** — called in: `status.go`, `destroy.go`, `logs.go`, `stop.go`, `agents.go`, `attach.go` (7 times)
- **`config.LoadFrom()`** — called in: `init.go`, `config_test.go`
- **`&config.Config{}` literals** — `init.go` (line 50), `config_test.go` (lines 14, 47)
- **Constants used in:** `init.go` (`GiteaAdminUser`, `GiteaAdminPass`), `status.go` (`GiteaAdminUser`, `GiteaAdminPass`), `manager.go` (`GiteaAdminUser`, `GiteaAdminPass`, `GiteaAdminEmail`), `server.go` (`GiteaAdminUser`)
- **`*config.Config` as parameter:** `manager.go` (`NewManager`), `spawner.go` (`NewSpawner`), `server.go` (`NewServer`)

### Existing tests
- `config_test.go`: `TestConfig_SaveAndLoad`, `TestConfig_GiteaURL`, `TestConfig_NoProjectFields`

## Acceptance Criteria

- [ ] `ConfigPort` interface defined with method(s) to resolve config values
- [ ] Defaults adapter returns hardcoded fallback values for all fields
- [ ] JSON file adapter reads from `~/.kiloforge/config.json` (and supports `Save()` for persistence)
- [ ] Env var adapter reads from `KF_*` environment variables (e.g., `KF_GITEA_PORT`, `KF_DATA_DIR`, `KF_API_TOKEN`, `KF_GITEA_IMAGE`, etc.)
- [ ] Flags adapter reads from Cobra flag values when explicitly set by the user
- [ ] Merger chains adapters in priority order: flags > env > JSON > defaults
- [ ] Previously hardcoded constants (`ContainerName`, `GiteaImage`, `GiteaAdminUser`, `GiteaAdminPass`, `GiteaAdminEmail`) are now configurable fields resolved through the adapter chain
- [ ] All 10 call sites updated to use the new resolution mechanism
- [ ] `Config` struct retained as the resolved value object (not an interface — the port is for _sources_)
- [ ] Existing tests updated and new tests added for each adapter and the merger
- [ ] `config.Save()` still writes to JSON file (only the JSON adapter is writable)
- [ ] Zero-value fields in higher-priority adapters do not override lower-priority values (merger skips unset fields)

## Dependencies

None

## Out of Scope

- YAML or TOML config file support
- Config validation beyond type correctness
- Config file watching / hot reload
- Project-level config (future `kf add` track)
- CLI flag definitions themselves (those stay in `internal/cli/` — the flags adapter just reads their values)

## Technical Notes

### Proposed Architecture

```
                    ┌─────────────┐
                    │   Config    │  (resolved value object)
                    │   struct    │
                    └──────▲──────┘
                           │ produces
                    ┌──────┴──────┐
                    │   Merger    │  resolves: flags > env > json > defaults
                    └──────▲──────┘
                           │ chains
          ┌────────┬───────┼───────┬──────────┐
          │        │       │       │          │
     ┌────┴───┐┌───┴──┐┌──┴──┐┌──┴───┐
     │Defaults││ JSON ││ Env ││Flags │
     │Adapter ││Adapter││Adapter││Adapter│
     └────────┘└──────┘└─────┘└──────┘
```

### ConfigPort Interface

```go
// ConfigProvider supplies config values from a single source.
// Returns a Config with only the fields this source knows about.
// Zero-value fields mean "not set by this source."
type ConfigProvider interface {
    Load() (*Config, error)
}
```

### Config Struct (expanded)

```go
type Config struct {
    GiteaPort       int    `json:"gitea_port"`
    DataDir         string `json:"data_dir"`
    APIToken        string `json:"api_token,omitempty"`
    ComposeFile     string `json:"compose_file,omitempty"`
    ContainerName   string `json:"container_name,omitempty"`
    GiteaImage      string `json:"gitea_image,omitempty"`
    GiteaAdminUser  string `json:"gitea_admin_user,omitempty"`
    GiteaAdminPass  string `json:"gitea_admin_pass,omitempty"`
    GiteaAdminEmail string `json:"gitea_admin_email,omitempty"`
}
```

### Env Var Mapping

| Field | Env Var | Example |
|-------|---------|---------|
| GiteaPort | `KF_GITEA_PORT` | `4000` |
| DataDir | `KF_DATA_DIR` | `/opt/kiloforge` |
| APIToken | `KF_API_TOKEN` | `abc123` |
| ComposeFile | `KF_COMPOSE_FILE` | `/path/to/compose.yml` |
| ContainerName | `KF_CONTAINER_NAME` | `my-gitea` |
| GiteaImage | `KF_GITEA_IMAGE` | `gitea/gitea:1.21` |
| GiteaAdminUser | `KF_GITEA_ADMIN_USER` | `admin` |
| GiteaAdminPass | `KF_GITEA_ADMIN_PASS` | `s3cret` |
| GiteaAdminEmail | `KF_GITEA_ADMIN_EMAIL` | `admin@example.com` |

### Merger Logic

The merger iterates adapters in reverse priority (defaults first, flags last). For each adapter, it calls `Load()` and overlays non-zero fields onto the accumulator. This means higher-priority sources win for any field they set.

### Call Site Migration

Current `config.Load()` calls become `config.Resolve(opts...)` or similar, where opts can inject the flags adapter. Commands that don't have flags just use `config.Resolve()` which chains defaults > JSON > env. The `init` command passes flag values via the flags adapter.

---

_Generated by conductor-track-generator from prompt: "Refactor config package to use port/adapter pattern with env var support, merger, defaults, env, and flags adapters"_
