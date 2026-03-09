# Backend Adapter Layer Findings

**Track:** arch-review_20260310040000Z
**Phase:** 2 — Backend Adapter Layer Review

## 1. CLI Thin-Adapter Compliance

### 1.1 Critical Violations

| # | Severity | File | Lines | Issue |
|---|----------|------|-------|-------|
| A1 | Critical | `cli/implement.go` | 411 total | Contains track validation, consent flow, worktree acquisition, completion callbacks, board state transitions — all business logic that belongs in services |
| A2 | High | `cli/skills.go` | 275 total | Contains skill config updates, GitHub release checking, version comparison, installation flow — complex workflows not delegated to services |

### 1.2 Minor Violations

| # | Severity | File | Issue |
|---|----------|------|-------|
| A3 | Medium | `cli/push.go:93-143` | Git fetch/ahead-behind check logic should be in a service |
| A4 | Medium | `cli/add.go:199-223` | SSH key discovery + selection logic should be a service |
| A5 | Low | `cli/daemon.go` | Process management could be extracted |

### 1.3 Compliant Files (20+)
`serve.go`, `dashboard.go`, `status.go`, `cost.go`, `destroy.go`, `init.go`, `logs.go`, `agents.go`, `up.go`, `down.go`, `projects.go`, `pool.go`, `escalated.go`, `attach.go`, `stop.go`, `prompt.go`, `sync.go`, `runtime.go`, `root.go`, `db.go`, `version.go` — all good.

### 1.4 Flag/Arg Parsing Consistency
- Consistent `flagXxx` global variable pattern across commands
- Consistent `cobra.ExactArgs()` / `cobra.MaximumNArgs()` validation
- Inconsistent stdin handling: `readLineCtx()` (context-aware) vs `bufio.NewScanner()` (context-unaware)

## 2. Schema-First API Compliance

### 2.1 OpenAPI Alignment — EXCELLENT
- **40 operations** defined in `openapi.yaml`, all implemented in `api_handler.go`
- Compile-time assertion: `var _ gen.StrictServerInterface = (*APIHandler)(nil)`
- `oapi-codegen` v2.6.0 strict handler pattern — no hand-written routes for API operations
- **Zero discrepancies** between spec and implementation

### 2.2 Non-OpenAPI Routes (Intentional)
9 routes exist in code but NOT in OpenAPI — all intentionally excluded:

| Route | Reason |
|-------|--------|
| `POST /webhook` | Gitea webhook intake |
| `GET /ws/agent/{id}` | WebSocket (covered in AsyncAPI) |
| `GET /api/badges/*` (3 routes) | SVG binary output |
| `GET/PUT /api/tour`, `GET /api/tour/demo-board` | Tour UI state |
| `GET /events` | SSE (covered in AsyncAPI) |

### 2.3 AsyncAPI Coverage — GOOD
`asyncapi.yaml` documents SSE channels (11 event types) and webhook intake (4 event types).

## 3. Persistence Layer

### 3.1 SQL Injection — SAFE
All SQL queries use parameterized `?` placeholders. No string interpolation in SQL.

### 3.2 Migration Pattern — GOOD
Goose-based versioned migrations with embedded SQL files. Legacy JSON-to-SQLite bridge is one-time-only.

### 3.3 Unhandled db.Exec() Errors — HIGH SEVERITY

| # | Severity | File | Lines | Issue |
|---|----------|------|-------|-------|
| A6 | High | `agent_store.go` | 55-64 | `AddAgent()` ignores Exec error — agent state not persisted |
| A7 | High | `agent_store.go` | 105-112 | `UpdateStatus()` ignores Exec error — status updates silently lost |
| A8 | High | `quota_store.go` | 22-35 | `RecordUsage()` ignores Exec error — quota tracking lost |
| A9 | High | `trace_store.go` | 61-107 | Multiple `Record()` Exec calls ignore errors — traces incomplete |
| A10 | High | `migrate_json.go` | 68 | `migrateConfig()` ignores Exec error — config migration silently fails |

### 3.4 Missing Domain Sentinel Errors

| # | Severity | File | Issue |
|---|----------|------|-------|
| A11 | Medium | `agent_store.go:67-82` | `FindAgent()` doesn't return `domain.ErrAgentNotFound` |
| A12 | Medium | `project_store.go:24-38` | `Get()` returns `(zero, false)` instead of sentinel error |
| A13 | Medium | `pr_tracking_store.go:23-40` | `LoadPRTracking()` doesn't return `domain.ErrPRTrackingNotFound` |

### 3.5 Remaining JSON Persistence
None — all migrated to SQLite. Legacy bridge is one-time import only.

## 4. Concurrency and Lifecycle

### 4.1 Multiple Output Relay Goroutines on Resume — CRITICAL

| # | Severity | File | Lines | Issue |
|---|----------|------|-------|-------|
| A14 | Critical | `api_handler.go` | 388, 495, 1486, 1620, 1878 | When agent is resumed, new `StartStructuredRelay()` goroutine spawned WITHOUT stopping previous one. Two goroutines consuming from same channel = dropped messages, panic risk |

**Scenario:** Agent spawned → relay A starts → agent stopped → agent resumed → relay B starts → both read from `ia.Output` → contention.

### 4.2 Potential Double-Close Panic

| # | Severity | File | Lines | Issue |
|---|----------|------|-------|-------|
| A15 | Medium | `sdk_client.go` | 197-198 | `Close()` calls `close(s.output)` and `close(s.done)` without sync.Once protection. Double-close panics if called concurrently (monitor goroutine + StopAgent handler) |

### 4.3 Context Lifecycle Issues

| # | Severity | File | Lines | Issue |
|---|----------|------|-------|-------|
| A16 | Medium | `ws/session.go` | 60 | WebSocket sessions use `context.Background()` instead of inheriting from request context. No graceful shutdown coordination |
| A17 | Medium | `ws/session.go` | 91-100 | `BroadcastToAgent` takes stale session snapshot — can write to cancelled context after disconnect |

### 4.4 OutputRelay Not Tracked

| # | Severity | File | Issue |
|---|----------|------|-------|
| A18 | Medium | `ws/session.go:155-165` | Relay goroutine not tracked for explicit cleanup — only terminates when output channel closes |

## Summary

| Category | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| CLI Thin-Adapter | 1 | 1 | 2 | 1 |
| Schema-First API | 0 | 0 | 0 | 0 |
| Persistence | 0 | 5 | 3 | 0 |
| Concurrency | 1 | 0 | 4 | 0 |
| **Total** | **2** | **6** | **9** | **1** |

### Track Recommendations
1. **fix-relay-goroutine-leak** — Track and stop relay goroutines before spawning new ones on resume. Add sync.Once to Close().
2. **fix-sqlite-error-handling** — Check and propagate all db.Exec() return errors across all stores.
3. **refactor-cli-implement** — Extract ImplementService from cli/implement.go (track validation, consent, worktree, callbacks).
4. **refactor-cli-skills** — Extract SkillsService from cli/skills.go (update, install, list workflows).
5. **fix-store-sentinel-errors** — Return domain sentinel errors from all store Find/Get methods.
6. **fix-ws-context-lifecycle** — Inherit request context in WebSocket sessions, add graceful shutdown.
