# Architectural Review — Prioritized Findings Summary

**Track:** arch-review_20260310040000Z
**Date:** 2026-03-10

## Executive Summary

Kiloforge has grown from a simple CLI relay to a full-stack platform with 19 adapter packages, 7 services, 40 REST endpoints, WebSocket/SSE real-time communication, SQLite persistence, and a React dashboard. Despite this velocity, the codebase maintains **strong schema-first API compliance** (zero drift), **excellent frontend type safety** (zero `any` types), and **consistent TanStack Query patterns**. The main risks are: a critical concurrency bug in agent resume, widespread unhandled database errors, service-layer boundary violations, and a significant frontend test coverage gap.

---

## All Findings by Severity

### Critical (3)

| ID | Category | Location | Description | Recommendation |
|----|----------|----------|-------------|----------------|
| A14 | Concurrency | `adapter/rest/api_handler.go:388,495` | **Multiple relay goroutines on agent resume.** When an agent is resumed, a new `StartStructuredRelay()` goroutine is spawned without stopping the previous one. Two goroutines consuming from the same channel causes dropped messages and panic risk. | Track relay goroutines; stop old relay before starting new one. Add `sync.Once` to `SDKSession.Close()`. |
| C9 | Architecture | `adapter/rest/api_handler.go:26` | **REST handler imports concrete service layer.** `api_handler.go` depends on `service.NativeBoardService`, `service.AddProjectOpts`, `service.DiscoverTracks` — adapters should depend on port interfaces. | Create port interfaces for board service, project operations, and track discovery. |
| A1 | Architecture | `adapter/cli/implement.go` (411 lines) | **CLI command contains business logic.** Track validation, consent flow, worktree acquisition, completion callbacks, board state transitions — all belong in services. | Extract `ImplementService` with focused methods. |

### High (11)

| ID | Category | Location | Description |
|----|----------|----------|-------------|
| A6-A10 | Persistence | `agent_store.go`, `quota_store.go`, `trace_store.go`, `migrate_json.go` | **5 unhandled `db.Exec()` errors.** Silent write failures in AddAgent, UpdateStatus, RecordUsage, Record (traces), migrateConfig. Data silently lost. |
| C1-C3 | Architecture | `service/project_service.go`, `service/board_service.go` | **3 interfaces defined in service instead of port.** `ProjectGiteaClient`, `ProjectStoreWriter`, `NativeBoardStore` should be in `port/`. |
| C6 | Architecture | `service/project_service.go` | **Error type duplication.** Custom `ProjectExistsError`/`ProjectNotFoundError` structs duplicate sentinel errors in `domain/errors.go`. |
| C10-C11 | Architecture | `adapter/rest/server.go:29`, `adapter/cli/runtime.go:9` | **Server and CLI runtime import concrete services.** Should depend on port interfaces. |
| A2 | Architecture | `adapter/cli/skills.go` (275 lines) | **CLI command contains complex workflows.** Skill update, installation, config logic not delegated. |
| F-T1 | Testing | `hooks/useAgentWebSocket.ts` (200 lines) | **Untested critical hook.** WebSocket reconnection, exponential backoff, message parsing — bugs invisible without tests. |
| F-T2 | Testing | `pages/ProjectPage.tsx` (361 lines) | **Untested main page.** 10+ hooks, cascading failure risk. |
| F-T3 | Testing | `components/KanbanBoard.tsx` (204 lines) | **Untested complex component.** Drag-and-drop, state machine for confirmations. |

### Medium (16)

| ID | Category | Location | Description |
|----|----------|----------|-------------|
| A11-A13 | Persistence | `agent_store.go`, `project_store.go`, `pr_tracking_store.go` | **3 stores don't return domain sentinel errors.** Find/Get methods return custom formats instead of `domain.ErrXxxNotFound`. |
| A15 | Concurrency | `sdk_client.go:197-198` | **Potential double-close panic.** `Close()` calls `close(channel)` without `sync.Once`. |
| A16 | Concurrency | `ws/session.go:60` | **WebSocket sessions use `context.Background()`.** No graceful shutdown coordination. |
| A17 | Concurrency | `ws/session.go:91-100` | **Stale session snapshot in broadcast.** Can write to cancelled context after disconnect. |
| A18 | Concurrency | `ws/session.go:155-165` | **OutputRelay goroutine not tracked.** Only terminates when channel closes — not explicitly stoppable. |
| C4 | Architecture | `service/track_service.go` | **Service uses only stdlib.** Reads filesystem directly instead of through a port. |
| C5 | Architecture | `service/project_service.go:5-11` | **Service uses `os/exec` directly.** Git commands should go through `port.GitRunner`. |
| C7-C8 | Architecture | `service/track_service.go`, `service/agent_service.go` | **Business entities in service layer.** `TrackEntry`, `TrackDetail`, `ProgressCount`, `EscalatedItem` belong in `domain/`. |
| A3 | Architecture | `cli/push.go:93-143` | **Git workflow logic in CLI.** Fetch/ahead-behind should be a service. |
| A4 | Architecture | `cli/add.go:199-223` | **SSH key discovery in CLI.** Should be a service. |
| F1 | Architecture | `pages/ProjectPage.tsx` (361 lines) | **Overloaded container component.** Should split into sub-containers. |
| F-T4-T8 | Testing | 5 hooks/pages | **Untested high-priority hooks.** useProjects, useBoard, AgentDetailPage, useSSE, OverviewPage. |

### Low (8)

| ID | Category | Location | Description |
|----|----------|----------|-------------|
| N1 | Naming | `rest/gen/server.gen.go` | **Mixed-case acronyms in generated code.** `AgentId`, `TrackId`, `RemoteUrl` etc. — oapi-codegen doesn't follow Go conventions. |
| N2 | Naming | `TrackList.module.css:40` | **Kebab-case CSS class.** `.in-progress` should be `.inProgress`. |
| F2 | Architecture | `components/AgentCard.tsx:17` | **Presentational component calls `useTracks()`.** Should receive data via props. |
| F3 | Architecture | `pages/AgentDetailPage.tsx:47-87` | **Mixed data fetching.** Raw `fetch()` alongside TanStack Query (justified but inconsistent). |
| F4 | Architecture | `useConsent.ts`, `useSkillsPrompt.ts`, `useSetupPrompt.ts` | **Prompt hook pattern duplication.** Could use shared factory. |
| F5 | Types | `types/api.ts` | **Missing API error response type.** |
| F7 | Architecture | `api/errorToast.ts` | **Global mutable state for toast.** Could use QueryClient MutationCache. |
| RBAC | Dead Code | N/A | **No RBAC implementation.** Memory references activity-based RBAC but only boolean consent exists. |

---

## Findings by Category

### Architecture Violations: 14 findings
The most common issue. Service layer defines its own interfaces instead of using ports. REST handler and CLI layer import concrete services. Two CLI commands contain substantial business logic.

### Concurrency/Lifecycle: 6 findings
One critical bug (relay goroutine leak on resume), one medium bug (double-close panic), and four medium lifecycle management gaps.

### Persistence: 8 findings
Five unhandled write errors (high severity — data silently lost). Three stores don't return domain sentinel errors.

### Test Coverage: 8 findings
Frontend has 8.8% test coverage (5/57 files). All critical/high test gaps are in hooks with complex logic and main pages.

### Naming: 2 findings
Generated Go code has mixed-case acronyms. One CSS class uses kebab-case.

---

## Schema-First API Compliance — EXCELLENT
Zero discrepancies between OpenAPI spec and implementation. All 40 operations have handlers with compile-time verification. AsyncAPI covers SSE and webhook channels. No remediation needed.

## Frontend Type Safety — EXCELLENT
Zero `any` types. 21 comprehensive interfaces. Consistent optional chaining and nullish coalescing. No remediation needed beyond minor error type addition.

---

## Recommended Follow-Up Tracks

### Priority 1 — Fix Bugs
1. **fix-relay-goroutine-leak** — Track and stop relay goroutines before spawning new ones on agent resume. Add sync.Once to SDKSession.Close(). Fix stale session broadcast. (Critical)
2. **fix-sqlite-error-handling** — Check and propagate all db.Exec() return errors across all 5 store files. (High)

### Priority 2 — Architecture Alignment
3. **refactor-port-interfaces** — Move service-local interfaces (ProjectGiteaClient, ProjectStoreWriter, NativeBoardStore) to port/. Create port interfaces for services used by REST handlers. (High)
4. **refactor-error-handling** — Standardize on domain sentinel errors. Remove duplicate error types from service layer. Update stores to return domain errors. (High)
5. **refactor-cli-implement** — Extract ImplementService from cli/implement.go (validation, consent, worktree, callbacks). (High)

### Priority 3 — Test Coverage
6. **frontend-test-hooks** — Tests for useAgentWebSocket, useSSE, useProjects, useBoard (critical hooks). (High)
7. **frontend-test-pages** — Tests for ProjectPage, AgentDetailPage, OverviewPage. (Medium)

### Priority 4 — Cleanup
8. **refactor-domain-types** — Move TrackEntry/TrackDetail/EscalatedItem from service to domain. (Medium)
9. **refactor-cli-skills** — Extract SkillsService from cli/skills.go. (Medium)
10. **refactor-project-page** — Split ProjectPage into sub-containers. (Medium)
11. **fix-naming-conventions** — Fix CSS kebab-case class. Configure oapi-codegen for Go acronym conventions if possible. (Low)

---

## Cross-Reference

| Document | Phase | Content |
|----------|-------|---------|
| `findings-backend-core.md` | 1 | Core layer dependency direction, service patterns, domain completeness |
| `findings-backend-adapters.md` | 2 | CLI compliance, schema-first API, persistence, concurrency |
| `findings-frontend.md` | 3 | Architecture, hooks, type safety, test coverage |
| `findings-summary.md` (this file) | 4 | Aggregated findings, priorities, track recommendations |
