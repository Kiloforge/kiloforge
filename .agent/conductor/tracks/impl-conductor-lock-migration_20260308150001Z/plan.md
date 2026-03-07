# Implementation Plan: Migrate Conductor Skills to Use crelay Lock API

**Track ID:** impl-conductor-lock-migration_20260308150001Z

## Phase 1: Lock Helper Functions (3 tasks)

### Task 1.1: Design lock helper shell functions
- [x] `is_relay_running()` — health check on relay URL
- [x] `acquire_lock(scope, timeout)` — try HTTP, fall back to mkdir
- [x] `release_lock(scope)` — release via HTTP or rm -rf based on mode
- [x] `start_heartbeat(scope)` — background heartbeat loop, return PID
- [x] `stop_heartbeat(pid)` — kill background heartbeat

### Task 1.2: Update conductor-developer SKILL.md
- [x] Replace mkdir lock acquire with `acquire_lock` helper
- [x] Replace `rm -rf` release with `release_lock` helper
- [x] Add heartbeat during rebase + verification + merge section
- [x] Update `--auto-merge` to use HTTP long-poll timeout when relay available
- [x] Keep mkdir fallback for when relay is not running
- [x] Document `CRELAY_RELAY_URL` env var

### Task 1.3: Update conductor-track-generator SKILL.md
- [x] Replace mkdir lock acquire with `acquire_lock` helper
- [x] Replace `rm -rf` release with `release_lock` helper
- [x] Add heartbeat during rebase + merge section
- [x] Keep mkdir fallback

## Phase 2: Testing and Validation (4 tasks)

### Task 2.1: Test HTTP lock path
- [x] Start relay server with lock service
- [x] Run conductor-developer merge flow — verify HTTP lock acquired/released
- [x] Run conductor-track-generator merge flow — verify HTTP lock acquired/released
- [x] Verify heartbeat keeps lock alive during long operations

### Task 2.2: Test mkdir fallback path
- [x] Stop relay server
- [x] Run conductor-developer merge flow — verify falls back to mkdir
- [x] Run conductor-track-generator merge flow — verify falls back to mkdir
- [x] Verify behavior is identical to pre-migration

### Task 2.3: Test crash recovery
- [x] Acquire HTTP lock, kill the holder process
- [x] Verify lock auto-expires after TTL
- [x] Verify next acquirer gets the lock without manual intervention

### Task 2.4: Test concurrent agents
- [x] Two developer agents attempt merge simultaneously
- [x] Verify one acquires, other blocks
- [x] Verify second acquires after first releases
- [x] Test with both HTTP and mkdir fallback modes

## Phase 3: Documentation (2 tasks)

### Task 3.1: Update skill documentation
- [x] Both skills document the dual-mode lock mechanism
- [x] Document `CRELAY_RELAY_URL` configuration
- [x] Document TTL and heartbeat behavior
- [x] Document fallback behavior clearly

### Task 3.2: Update project memory
- [x] Update MEMORY.md to reflect new lock mechanism
- [x] Note: HTTP lock preferred, mkdir fallback retained

---

**Total: 9 tasks across 3 phases**
