# Implementation Plan: Refactor CLI Commands to Thin Adapters

**Track ID:** refactor-cli-thin-adapters_20260310010000Z

## Phase 1: Service Layer Foundations

- [ ] Task 1.1: Create `AgentService` in `core/service/agent_service.go` with `ListAgents()`, `GetAgent()`, `StopAgent()`, `GetCostReport()`, `GetEscalated()`
- [ ] Task 1.2: Add `ListProjects()` and `GetProject()` to `ProjectService`
- [ ] Task 1.3: Add unit tests for new `AgentService` methods
- [ ] Task 1.4: Add unit tests for new `ProjectService` query methods

## Phase 2: Shared CLI Runtime

- [ ] Task 2.1: Create `CLIRuntime` struct in `adapter/cli/runtime.go` — shared service graph construction
- [ ] Task 2.2: `CLIRuntime` opens SQLite, constructs stores, constructs all services
- [ ] Task 2.3: Add `Close()` method for cleanup

## Phase 3: Migrate CLI Commands

- [ ] Task 3.1: Refactor `projects.go` — use `rt.ProjectService.ListProjects()`
- [ ] Task 3.2: Refactor `agents.go` — use `rt.AgentService.ListAgents()`
- [ ] Task 3.3: Refactor `stop.go` — use `rt.AgentService.StopAgent()`
- [ ] Task 3.4: Refactor `attach.go` — use `rt.AgentService.GetAgent()`
- [ ] Task 3.5: Refactor `status.go` — use `rt.AgentService`, remove concrete store type signatures
- [ ] Task 3.6: Refactor `cost.go` — use `rt.AgentService.GetCostReport()`, remove concrete type signatures
- [ ] Task 3.7: Refactor `escalated.go` — use `rt.AgentService.GetEscalated()`
- [ ] Task 3.8: Refactor `add.go` — delegate to `rt.ProjectService.AddProject()`, remove inline business logic
- [ ] Task 3.9: Refactor `push.go` — use `rt.ProjectService.GetProject()` + push logic in service
- [ ] Task 3.10: Refactor `sync.go` — use `rt.BoardService`, remove direct store construction

## Phase 4: Cleanup

- [ ] Task 4.1: Verify no CLI command imports `persistence/sqlite` directly (only `runtime.go`)
- [ ] Task 4.2: Verify no CLI command has concrete store types in function signatures
- [ ] Task 4.3: Remove any dead helper functions that were inlined in CLI commands

## Phase 5: Verification

- [ ] Task 5.1: `make build` succeeds
- [ ] Task 5.2: `make test` passes
- [ ] Task 5.3: Smoke test — `kf projects`, `kf agents`, `kf status`, `kf cost`, `kf stop` produce correct output
- [ ] Task 5.4: Smoke test — `kf add` registers a project correctly via the service
