# Specification: Migrate Conductor Skills to Use crelay Lock API

**Track ID:** impl-conductor-lock-migration_20260308150001Z
**Type:** Chore
**Created:** 2026-03-08T15:00:01Z
**Status:** Draft

## Summary

Update conductor-developer and conductor-track-generator skills to use the crelay HTTP lock API when the relay server is available, with automatic fallback to the existing mkdir-based mechanism when it is not. No breaking changes to default behavior.

## Context

The crelay relay server now provides an HTTP-based scoped lock service (track `impl-lock-service_20260308150000Z`) with TTL, heartbeat, and crash recovery. Conductor skills currently use `mkdir` at `$(git rev-parse --git-common-dir)/merge.lock` for merge serialization. This migration:
- Prefers the HTTP lock API when relay is reachable
- Falls back to mkdir when relay is not running (e.g., standalone conductor usage)
- Adds heartbeat during long merge operations
- Improves crash recovery (TTL auto-expires stale locks)

## Codebase Analysis

### Files to modify

- **`~/.claude/skills/conductor-developer/SKILL.md`** — merge lock acquisition (lines ~433-469), release, and `--auto-merge` retry loop
- **`~/.claude/skills/conductor-track-generator/SKILL.md`** — merge lock acquisition (lines ~400-417) and release

### Current lock pattern in both skills

```bash
# Acquire
LOCK_DIR="$(git rev-parse --git-common-dir)/merge.lock"
if ! mkdir "$LOCK_DIR" 2>/dev/null; then
  echo "MERGE LOCK HELD"
  exit 1
fi

# Release
rm -rf "$LOCK_DIR"
```

### Proposed pattern (with fallback)

```bash
# Try HTTP lock first, fall back to mkdir
RELAY_URL="${CRELAY_RELAY_URL:-http://localhost:3001}"
HOLDER="$(basename $(pwd))"  # e.g., "developer-1", "track-generator-1"

acquire_lock() {
  # Try HTTP API
  if curl -sf -X POST "$RELAY_URL/api/locks/merge/acquire" \
    -H "Content-Type: application/json" \
    -d "{\"holder\": \"$HOLDER\", \"ttl_seconds\": 120, \"timeout_seconds\": $1}" \
    -o /dev/null 2>/dev/null; then
    echo "HTTP lock acquired"
    LOCK_MODE="http"
    return 0
  fi
  # Fall back to mkdir
  LOCK_DIR="$(git rev-parse --git-common-dir)/merge.lock"
  if mkdir "$LOCK_DIR" 2>/dev/null; then
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $HOLDER" > "$LOCK_DIR/info"
    echo "mkdir lock acquired (relay unavailable)"
    LOCK_MODE="mkdir"
    return 0
  fi
  return 1
}

release_lock() {
  if [ "$LOCK_MODE" = "http" ]; then
    curl -sf -X DELETE "$RELAY_URL/api/locks/merge" \
      -H "Content-Type: application/json" \
      -d "{\"holder\": \"$HOLDER\"}" 2>/dev/null || true
  else
    rm -rf "$(git rev-parse --git-common-dir)/merge.lock"
  fi
}
```

## Acceptance Criteria

- [ ] conductor-developer SKILL.md updated with HTTP lock acquire/release
- [ ] conductor-track-generator SKILL.md updated with HTTP lock acquire/release
- [ ] Automatic fallback to mkdir when relay is unreachable
- [ ] `CRELAY_RELAY_URL` env var for configuring relay endpoint (default: `http://localhost:3001`)
- [ ] Lock holder derived from worktree folder name (e.g., `developer-1`)
- [ ] Heartbeat sent during long operations (rebase, test verification)
- [ ] TTL set to 120 seconds with heartbeat every 30 seconds during merge
- [ ] `--auto-merge` uses HTTP long-poll timeout instead of `sleep 10` loop when relay available
- [ ] Release on any failure path (same as before, just via HTTP)
- [ ] Skills document both lock modes in their instructions
- [ ] No breaking changes: skills work identically when relay is not running

## Dependencies

- impl-lock-service_20260308150000Z (provides the HTTP lock API)

## Blockers

None

## Conflict Risk

- **LOW** — Modifies skill `.md` files only (`~/.claude/skills/conductor-developer/SKILL.md` and `~/.claude/skills/conductor-track-generator/SKILL.md`). These are not in the crelay repo — they're user-level skill configurations. No conflict with any pending tracks.

## Out of Scope

- Adding lock scopes beyond "merge" (future — the API supports arbitrary scopes)
- Lock visualization in dashboard (already available via `GET /api/locks`)
- Removing mkdir fallback entirely (keep for backward compatibility)
- Modifying other skills (conductor-reviewer, conductor-manage, etc.)

## Technical Notes

### Heartbeat during merge

The merge operation can take a long time (rebase + full test suite). The skill should send heartbeats:

```bash
# Start heartbeat in background during merge operations
heartbeat_loop() {
  while true; do
    sleep 30
    curl -sf -X POST "$RELAY_URL/api/locks/merge/heartbeat" \
      -H "Content-Type: application/json" \
      -d "{\"holder\": \"$HOLDER\", \"ttl_seconds\": 120}" 2>/dev/null || true
  done
}

# In merge section:
heartbeat_loop &
HEARTBEAT_PID=$!
# ... rebase, verify, merge ...
kill $HEARTBEAT_PID 2>/dev/null
```

### Relay detection

Simple health check to determine if HTTP locking is available:

```bash
is_relay_running() {
  curl -sf "$RELAY_URL/health" -o /dev/null 2>/dev/null
}
```

### Error handling

- If HTTP acquire fails with non-409 error (server error, network error): fall back to mkdir
- If HTTP release fails: log warning but don't fail the operation
- If heartbeat fails: log warning, lock will auto-expire via TTL (acceptable)

---

_Generated by conductor-track-generator_
