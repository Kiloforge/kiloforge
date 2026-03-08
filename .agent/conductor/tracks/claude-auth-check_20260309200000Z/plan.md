# Implementation Plan: Claude CLI Authentication Check Before Agent Spawning

**Track ID:** claude-auth-check_20260309200000Z

## Phase 1: Auth Check Implementation

- [x] Task 1.1: Create `backend/internal/adapter/prereq/auth.go` — `CheckClaudeAuth()` function with probe command, timeout, and stderr parsing
- [x] Task 1.2: Add `AuthError` type with `Message` and `Hint` fields, implement `error` interface
- [x] Task 1.3: Add process-level caching via `sync.Once` — `CheckClaudeAuthCached()` wrapper
- [x] Task 1.4: Write unit tests for `auth.go` — mock exec, test auth error detection, test timeout handling, test cache behavior

## Phase 2: Spawner Integration

- [x] Task 2.1: Add `checkAuth()` method to `Spawner` — calls `CheckClaudeAuthCached()`
- [x] Task 2.2: Update `SpawnDeveloper()`, `SpawnReviewer()`, `SpawnInteractive()` to call `checkAuth()` before `checkQuota()`
- [x] Task 2.3: Update `kf init` prereq check — call `CheckClaudeAuth()` as a warning (not blocking) alongside existing checks

## Phase 3: REST API and CLI Integration

- [x] Task 3.1: REST spawn endpoints (`POST /api/agents/*`, `POST /api/tracks/generate`, `POST /api/admin/run`) — return 401 with auth error message if `CheckClaudeAuthCached()` fails
- [x] Task 3.2: CLI `kf implement` — show auth error with login instructions before reaching skill/consent checks
- [x] Task 3.3: Add `GET /api/preflight` endpoint — returns combined auth + skills + consent status for dashboard pre-spawn checks

## Phase 4: Verification

- [ ] Task 4.1: `go test ./...` passes
- [ ] Task 4.2: Manual test — spawn with logged-in Claude succeeds; spawn without auth shows clear error
