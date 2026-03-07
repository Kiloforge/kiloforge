# Implementation Plan: Migrate Conductor Skills to Use crelay Lock API

**Track ID:** impl-conductor-lock-migration_20260308150001Z

## Phase 1: Lock Helper Functions (3 tasks)

### Task 1.1: Design lock helper shell functions
- [ ] `is_relay_running()` — health check on relay URL
- [ ] `acquire_lock(scope, timeout)` — try HTTP, fall back to mkdir
- [ ] `release_lock(scope)` — release via HTTP or rm -rf based on mode
- [ ] `start_heartbeat(scope)` — background heartbeat loop, return PID
- [ ] `stop_heartbeat(pid)` — kill background heartbeat

### Task 1.2: Update conductor-developer SKILL.md
- [ ] Replace mkdir lock acquire with `acquire_lock` helper
- [ ] Replace `rm -rf` release with `release_lock` helper
- [ ] Add heartbeat during rebase + verification + merge section
- [ ] Update `--auto-merge` to use HTTP long-poll timeout when relay available
- [ ] Keep mkdir fallback for when relay is not running
- [ ] Document `CRELAY_RELAY_URL` env var

### Task 1.3: Update conductor-track-generator SKILL.md
- [ ] Replace mkdir lock acquire with `acquire_lock` helper
- [ ] Replace `rm -rf` release with `release_lock` helper
- [ ] Add heartbeat during rebase + merge section
- [ ] Keep mkdir fallback

## Phase 2: Testing and Validation (4 tasks)

### Task 2.1: Test HTTP lock path
- [ ] Start relay server with lock service
- [ ] Run conductor-developer merge flow — verify HTTP lock acquired/released
- [ ] Run conductor-track-generator merge flow — verify HTTP lock acquired/released
- [ ] Verify heartbeat keeps lock alive during long operations

### Task 2.2: Test mkdir fallback path
- [ ] Stop relay server
- [ ] Run conductor-developer merge flow — verify falls back to mkdir
- [ ] Run conductor-track-generator merge flow — verify falls back to mkdir
- [ ] Verify behavior is identical to pre-migration

### Task 2.3: Test crash recovery
- [ ] Acquire HTTP lock, kill the holder process
- [ ] Verify lock auto-expires after TTL
- [ ] Verify next acquirer gets the lock without manual intervention

### Task 2.4: Test concurrent agents
- [ ] Two developer agents attempt merge simultaneously
- [ ] Verify one acquires, other blocks
- [ ] Verify second acquires after first releases
- [ ] Test with both HTTP and mkdir fallback modes

## Phase 3: Documentation (2 tasks)

### Task 3.1: Update skill documentation
- [ ] Both skills document the dual-mode lock mechanism
- [ ] Document `CRELAY_RELAY_URL` configuration
- [ ] Document TTL and heartbeat behavior
- [ ] Document fallback behavior clearly

### Task 3.2: Update project memory
- [ ] Update MEMORY.md to reflect new lock mechanism
- [ ] Note: HTTP lock preferred, mkdir fallback retained

---

**Total: 9 tasks across 3 phases**
