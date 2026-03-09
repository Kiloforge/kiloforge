# Specification: Refactor CLI Commands to Thin Adapters with Shared Service Layer

**Track ID:** refactor-cli-thin-adapters_20260310010000Z
**Type:** Refactor
**Created:** 2026-03-10T01:00:00Z
**Status:** Draft

## Summary

Refactor all CLI commands to be thin adapters that dispatch to the service layer, matching the pattern already used by REST handlers. Currently, CLI commands bypass the service layer and access stores directly, duplicating business logic and creating inconsistencies. After this refactor, both CLI and API will share the same domain logic through services.

## Context

Product guideline #7 (newly added) mandates: "CLI commands and REST handlers are thin adapters that convert input into domain commands/queries and dispatch to the service layer. Adapters never access stores directly."

Currently, the codebase has a split personality:
- **REST handlers** (correct): parse request → call service → return response
- **CLI commands** (incorrect): parse flags → instantiate stores directly → inline business logic → format output

This causes:
- Duplicated business logic (e.g., `add.go` has 80+ lines of project registration logic that `ProjectService.AddProject()` already handles)
- Untestable CLI commands (tightly coupled to concrete implementations)
- Divergent error handling between CLI and API for the same operations

## Codebase Analysis

### CLI commands that need refactoring (10 files)

1. **`add.go`** — Inlines project cloning, Gitea repo creation, webhook setup. Should delegate to `ProjectService.AddProject()`
2. **`agents.go`** — Loads store directly. Needs `AgentService.ListAgents()`
3. **`attach.go`** — Loads store directly. Needs `AgentService.GetAgent()`
4. **`cost.go`** — Loads store, has helper functions with concrete store type signatures. Needs `AgentService.GetCostReport()`
5. **`escalated.go`** — Loads stores directly. Needs `AgentService.GetEscalated()`
6. **`projects.go`** — Loads store directly. Needs `ProjectService.ListProjects()`
7. **`push.go`** — Loads store directly. Needs `ProjectService.GetProject()` + push logic in service
8. **`status.go`** — Loads store, has concrete type in signatures. Needs `AgentService` queries
9. **`stop.go`** — Loads store, calls `HaltAgent()` directly. Needs `AgentService.StopAgent()`
10. **`sync.go`** — Loads stores directly, creates store. Already partially correct (uses `NativeBoardService`), but store construction should move

### Services that need to be created or extended

**New: `AgentService`** — Currently no service exists for agent operations:
- `ListAgents() ([]domain.AgentInfo, error)`
- `GetAgent(id string) (*domain.AgentInfo, error)`
- `StopAgent(id string) error`
- `GetCostReport() (*CostReport, error)`
- `GetEscalated() ([]EscalatedItem, error)`

**Extend: `ProjectService`** — Add missing query methods:
- `ListProjects() ([]domain.Project, error)`
- `GetProject(slug string) (*domain.Project, error)`

### The correct pattern (already working in `dashboard.go`)

`dashboard.go` shows how CLI should work:
1. Resolve config
2. Open SQLite DB
3. Construct stores from DB
4. Construct services with stores injected
5. Pass services to handlers
6. Services contain all business logic

## Acceptance Criteria

- [ ] All CLI commands dispatch to service methods — no direct store access
- [ ] No CLI command imports `persistence/sqlite` directly (only `cli/runtime.go` does)
- [ ] New `AgentService` created in `core/service/` with list, get, stop, cost, and escalated operations
- [ ] `ProjectService` extended with `ListProjects()` and `GetProject()` query methods
- [ ] CLI commands use port interfaces, not concrete store types, in function signatures
- [ ] All CLI commands construct their service graph via shared `CLIRuntime`: config → DB → stores → services → execute
- [ ] `make test` passes
- [ ] `make build` succeeds
- [ ] CLI commands produce the same output as before (no user-facing changes)

## Dependencies

- **cli-sqlite-migration_20260310005000Z** (completed) — CLI already uses SQLite

## Blockers

None.

## Conflict Risk

- HIGH — touches 10 CLI command files. This track should be prioritized and run when no other CLI-modifying tracks are in-progress.

## Out of Scope

- REST handler refactoring (already correct)
- Adding new CLI commands
- Changing CLI output format or behavior
- Modifying the SQLite store implementations

## Technical Notes

### Shared CLI runtime

```go
// backend/internal/adapter/cli/runtime.go
type CLIRuntime struct {
    DB             *sqlite.DB
    ProjectService *service.ProjectService
    AgentService   *service.AgentService
    BoardService   *service.NativeBoardService
    TrackService   *service.TrackService
    Config         port.Config
}

func NewCLIRuntime(cfg port.Config) (*CLIRuntime, error) {
    db, err := sqlite.Open(cfg.DataDir())
    if err != nil { return nil, fmt.Errorf("open database: %w", err) }
    projectStore := sqlite.NewProjectStore(db)
    agentStore := sqlite.NewAgentStore(db)
    boardStore := sqlite.NewBoardStore(db)
    return &CLIRuntime{
        DB:             db,
        ProjectService: service.NewProjectService(projectStore, giteaClient, serviceCfg),
        AgentService:   service.NewAgentService(agentStore),
        BoardService:   service.NewNativeBoardService(boardStore),
        Config:         cfg,
    }, nil
}

func (r *CLIRuntime) Close() error { return r.DB.Close() }
```

Each CLI command becomes:

```go
func runProjects(cmd *cobra.Command, args []string) error {
    rt, err := NewCLIRuntime(cfg)
    if err != nil { return err }
    defer rt.Close()
    projects, err := rt.ProjectService.ListProjects()
    if err != nil { return err }
    // format and print
}
```

---

_Generated by conductor-track-generator from prompt: "refactor CLI commands to thin adapters with shared service layer"_
