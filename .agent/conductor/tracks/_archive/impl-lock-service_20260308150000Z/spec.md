# Specification: HTTP-Based Scoped Lock Service in Relay Server

**Track ID:** impl-lock-service_20260308150000Z
**Type:** Feature
**Created:** 2026-03-08T15:00:00Z
**Status:** Draft

## Summary

Add a scoped lock manager with TTL/heartbeat to the relay server, exposed via REST API endpoints. Supports named lock scopes (e.g., "merge", "deploy", custom), blocking acquire with timeout via HTTP long-poll, explicit release, and automatic expiry on TTL without heartbeat. Crash-safe by design.

## Context

Conductor developer and track-generator agents currently use `mkdir`-based atomic locks at `$(git rev-parse --git-common-dir)/merge.lock` for merge serialization. This has several weaknesses:
- Stale locks survive process crashes (manual `rm -rf` required)
- No lock holder identity verification (just timestamp + branch name)
- No timeout or heartbeat — impossible to tell if holder is alive
- Only supports one scope ("merge") — no general-purpose locking
- Polling with `sleep 10` is wasteful and unobservable

An HTTP-based lock service in the relay server solves all of these while remaining backward-compatible (agents can fall back to mkdir when relay isn't running).

## Codebase Analysis

- **Relay server**: `internal/relay/server.go` — HTTP mux with `/webhook` and `/health` endpoints
- **Current lock**: mkdir-based in skill files only — no lock code in the Go codebase
- **Config**: relay runs on port 3001, accessible from all worktrees on localhost
- **State persistence**: JSON files in DataDir — proven pattern for new state

### Design constraints

- Lock service runs inside the existing relay server (no separate process)
- Must handle concurrent requests from multiple agent processes
- Must auto-expire locks when holder crashes (TTL + heartbeat)
- Must support long-poll for blocking acquire (avoid sleep loops)
- Dashboard (future) should be able to display lock state

## Acceptance Criteria

- [ ] `internal/lock/manager.go` — thread-safe in-memory lock manager with TTL support
- [ ] Lock scopes: arbitrary string names (e.g., "merge", "deploy", "worktree-pool")
- [ ] `Acquire(scope, holder, ttl)` — returns immediately if free, blocks up to timeout if held
- [ ] `Release(scope, holder)` — only the holder can release their own lock
- [ ] `Heartbeat(scope, holder)` — extends TTL, proves holder is alive
- [ ] Auto-expire: locks released automatically when TTL expires without heartbeat
- [ ] `GET /api/locks` — list all active locks with holder, scope, TTL remaining
- [ ] `POST /api/locks/:scope/acquire` — acquire lock (body: holder, ttl, timeout)
- [ ] `POST /api/locks/:scope/heartbeat` — extend lock TTL
- [ ] `DELETE /api/locks/:scope` — release lock (body: holder)
- [ ] Long-poll acquire: request blocks until lock available or timeout expires
- [ ] HTTP 200 on acquire success, 409 on timeout (lock not acquired), 404 on release of unheld lock
- [ ] Lock state persisted to `{DataDir}/locks.json` for crash recovery
- [ ] Unit tests: concurrent acquire/release, TTL expiry, heartbeat extension, holder validation
- [ ] Race detector passes: `go test -race ./internal/lock/...`
- [ ] All existing tests pass, build succeeds

## Dependencies

None

## Blockers

- **impl-conductor-lock-migration_20260308150001Z** — depends on this track for the lock API

## Conflict Risk

- **LOW** — Adds new `internal/lock/` package (no existing files modified) and new endpoints to relay server mux. The relay server modification is a single `mux.HandleFunc()` addition per endpoint — minimal merge conflict risk with other tracks.
- Pending `refactor-clean-arch` would move relay to `adapter/rest/` — if that runs first, add endpoints there instead.

## Out of Scope

- Migrating conductor skills to use this API (track 2)
- Distributed locking across multiple machines
- Lock priority or fairness queuing
- Dashboard visualization of locks (can use `GET /api/locks` endpoint)

## Technical Notes

### Lock manager design

```go
// internal/lock/manager.go

type Lock struct {
    Scope     string    `json:"scope"`
    Holder    string    `json:"holder"`     // e.g., "developer-1", "track-generator-1"
    AcquiredAt time.Time `json:"acquired_at"`
    ExpiresAt  time.Time `json:"expires_at"`
}

type Manager struct {
    mu       sync.Mutex
    locks    map[string]*Lock           // scope → lock
    waiters  map[string][]chan struct{}  // scope → waiting acquire channels
    dataDir  string
}

func New(dataDir string) *Manager
func (m *Manager) Acquire(ctx context.Context, scope, holder string, ttl time.Duration) error
func (m *Manager) Release(scope, holder string) error
func (m *Manager) Heartbeat(scope, holder string, ttl time.Duration) error
func (m *Manager) List() []Lock
func (m *Manager) startReaper(ctx context.Context)  // background goroutine to expire stale locks
```

### API design

```
POST /api/locks/:scope/acquire
  Body: {"holder": "developer-1", "ttl_seconds": 60, "timeout_seconds": 300}
  Response 200: {"scope": "merge", "holder": "developer-1", "expires_at": "..."}
  Response 409: {"error": "timeout waiting for lock", "current_holder": "developer-2"}

POST /api/locks/:scope/heartbeat
  Body: {"holder": "developer-1", "ttl_seconds": 60}
  Response 200: {"scope": "merge", "expires_at": "..."}
  Response 404: {"error": "lock not held by this holder"}

DELETE /api/locks/:scope
  Body: {"holder": "developer-1"}
  Response 200: {"released": true}
  Response 404: {"error": "lock not held by this holder"}

GET /api/locks
  Response 200: [{"scope": "merge", "holder": "developer-1", "acquired_at": "...", "expires_at": "..."}]
```

### Long-poll acquire

```go
func (m *Manager) Acquire(ctx context.Context, scope, holder string, ttl time.Duration) error {
    m.mu.Lock()
    if existing, ok := m.locks[scope]; !ok || time.Now().After(existing.ExpiresAt) {
        // Lock is free or expired — acquire immediately
        m.locks[scope] = &Lock{Scope: scope, Holder: holder, ...}
        m.mu.Unlock()
        return nil
    }
    // Lock held — register waiter channel
    ch := make(chan struct{}, 1)
    m.waiters[scope] = append(m.waiters[scope], ch)
    m.mu.Unlock()

    select {
    case <-ch:
        // Notified that lock is free — try to acquire
        return m.Acquire(ctx, scope, holder, ttl)
    case <-ctx.Done():
        return ErrTimeout
    }
}
```

### TTL reaper

Background goroutine checks for expired locks every second:
```go
func (m *Manager) startReaper(ctx context.Context) {
    ticker := time.NewTicker(time.Second)
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C:
            m.mu.Lock()
            for scope, lock := range m.locks {
                if time.Now().After(lock.ExpiresAt) {
                    delete(m.locks, scope)
                    m.notifyWaiters(scope)
                }
            }
            m.mu.Unlock()
        }
    }
}
```

### Persistence

Locks saved to `{DataDir}/locks.json` on acquire/release. On startup, loads persisted locks but only restores those whose TTL hasn't expired — automatically cleans up crash-orphaned locks.

---

_Generated by conductor-track-generator_
