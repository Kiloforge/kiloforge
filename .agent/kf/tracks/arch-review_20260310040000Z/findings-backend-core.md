# Backend Core Layer Findings

**Track:** arch-review_20260310040000Z
**Phase:** 1 — Backend Core Layer Review

## 1. Dependency Direction Violations

### 1.1 Port Layer — COMPLIANT
All port files correctly import only `domain/` and stdlib. No violations.

### 1.2 Service Layer — VIOLATIONS FOUND

| # | Severity | File | Issue |
|---|----------|------|-------|
| C1 | High | `service/project_service.go:17-23` | Defines `ProjectGiteaClient` interface locally instead of using port interface |
| C2 | High | `service/project_service.go:26-32` | Defines `ProjectStoreWriter` interface locally instead of using port interface |
| C3 | High | `service/board_service.go:10-14` | Defines `NativeBoardStore` interface locally instead of using port interface |
| C4 | Medium | `service/track_service.go` | Uses only stdlib imports — completely isolated from domain/port abstractions. Reads filesystem directly instead of through a port |
| C5 | Medium | `service/project_service.go:5-11` | Uses `os/exec` directly for git commands instead of `port.GitRunner` |

**Pattern:** Services define their own interfaces rather than depending on centralized port interfaces. This fragments the interface landscape — consumers looking at `port/` won't see the full set of abstractions.

**Recommendation:** Move `ProjectGiteaClient`, `ProjectStoreWriter`, `NativeBoardStore` to `port/`. Refactor `track_service.go` to use a port interface for file/track access.

### 1.3 Domain Layer — COMPLIANT (minor note)
Domain files import only stdlib. `domain_test.go` uses `domain` package import for testing, which is standard Go test pattern (not a real violation).

## 2. Service Layer Pattern Issues

### 2.1 Error Type Duplication

| # | Severity | File | Issue |
|---|----------|------|-------|
| C6 | High | `service/project_service.go` | Defines `ProjectExistsError` and `ProjectNotFoundError` as custom struct types, duplicating sentinel errors already in `domain/errors.go` (`ErrProjectExists`, `ErrProjectNotFound`) |

The REST handler (`api_handler.go`) checks these via type assertions (`errors.As`) instead of `errors.Is` with sentinel errors. This breaks the standard sentinel error pattern.

**Recommendation:** Remove custom error types from service, use `fmt.Errorf("project %q: %w", slug, domain.ErrProjectNotFound)` pattern instead.

### 2.2 Constructor Injection
All 7 services use constructor injection (`NewXxx(...)`) — compliant. Services receive interfaces.

## 3. Domain Layer Completeness

### 3.1 Sentinel Errors — 7 defined
`ErrProjectNotFound`, `ErrProjectExists`, `ErrAgentNotFound`, `ErrPRTrackingNotFound`, `ErrPoolExhausted`, `ErrGiteaUnreachable`, `ErrForbidden`

All entities are pure Go with only stdlib dependencies. No I/O in domain layer.

### 3.2 Missing Domain Types

| # | Severity | File | Issue |
|---|----------|------|-------|
| C7 | Medium | `service/track_service.go:22-83` | `TrackEntry`, `TrackDetail`, `ProgressCount` are business entities defined in service instead of domain |
| C8 | Low | `service/agent_service.go:77-83` | `EscalatedItem` is a business concept defined in service |

### 3.3 RBAC/Permissions — Dead Pattern
The memory references "Activity-based RBAC" but the codebase only has a boolean consent flag (`--dangerously-skip-permissions`). No actual RBAC implementation exists. The consent code in `adapter/persistence/sqlite/consent.go` is a simple gate, not a permission system.

## 4. Adapter Cross-Dependencies

### 4.1 Adapters Importing Concrete Service Layer — CRITICAL

10 adapter files import `kiloforge/internal/core/service` directly:

| # | Severity | File | Issue |
|---|----------|------|-------|
| C9 | Critical | `adapter/rest/api_handler.go:26` | Imports `service` — uses `service.NativeBoardService`, `service.AddProjectOpts`, `service.DiscoverTracks`, `service.GetTrackDetail` |
| C10 | Critical | `adapter/rest/server.go:29` | Imports `service` — constructs services directly |
| C11 | Critical | `adapter/cli/runtime.go:9` | Imports `service` — acts as service locator, constructs all concrete services |
| C12 | High | `adapter/cli/serve.go:21` | Imports `service` — constructs services |
| C13 | High | `adapter/cli/add.go:14` | Imports `service` — calls `service.NewProjectService()` |
| C14 | High | `adapter/cli/dashboard.go:19` | Imports `service` — constructs services |
| C15 | High | `adapter/cli/implement.go:22` | Imports `service` — uses `service.DiscoverTracks`, `service.GetTrackDetail` |
| C16 | Medium | `adapter/cli/cost.go:10` | Imports `service` — accepts concrete `service.AgentService` |
| C17 | Medium | `adapter/cli/status.go:13` | Imports `service` — uses concrete service types |
| C18 | Medium | `adapter/dashboard/watcher.go:8` | Imports `service` — uses `service.DiscoverTracks` |

**Note:** CLI commands constructing services is a common pragmatic pattern — the CLI layer acts as the composition root. However, `api_handler.go` and `dashboard/watcher.go` importing service directly is a clear violation. REST handlers should depend on port interfaces, not concrete services.

### 4.2 Adapter-to-Adapter Dependencies
Cross-adapter imports are widespread (REST depends on agent, config, ws, etc.; CLI depends on nearly everything). This is acceptable at the composition root level but indicates the CLI/REST layers are doing too much wiring.

## Summary

| Category | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| Dependency Direction | 3 | 5 | 5 | 1 |
| **Total** | **3** | **5** | **5** | **1** |

### Track Recommendations
1. **refactor-port-interfaces** — Move service-local interfaces to port/, create port interfaces for services used by REST handlers
2. **refactor-error-handling** — Standardize on domain sentinel errors, remove duplicate error types
3. **refactor-domain-types** — Move TrackEntry/TrackDetail/EscalatedItem to domain
4. **refactor-track-service-io** — Abstract filesystem access behind a port interface
