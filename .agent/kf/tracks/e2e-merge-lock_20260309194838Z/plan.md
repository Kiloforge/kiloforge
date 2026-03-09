# Implementation Plan: E2E Tests — Merge Lock Coordination

**Track ID:** e2e-merge-lock_20260309194838Z

## Phase 1: Acquire Tests

- [ ] Task 1.1: Acquire happy path — POST to `/api/locks/{scope}/acquire` with holder and TTL, verify 200 response with lock info (scope, holder, expiry), verify lock appears in `GET /api/locks` list
- [ ] Task 1.2: Acquire with custom TTL — acquire lock with custom `ttl_seconds` value (e.g., 30), verify the returned expiry reflects the requested TTL, verify lock info displays correct remaining time
- [ ] Task 1.3: Acquire returns lock info — verify response body contains `scope`, `holder`, `ttl_seconds`, `expires_at` fields with correct types and values, verify `expires_at` is in the future

## Phase 2: Heartbeat Tests

- [ ] Task 2.1: Heartbeat extends TTL — acquire lock with short TTL (5s), wait 2s, send heartbeat, verify `expires_at` in response is later than original expiry
- [ ] Task 2.2: Heartbeat response — verify heartbeat returns 200 with updated lock info including new `expires_at`, verify holder matches the original acquire holder
- [ ] Task 2.3: Heartbeat timing — acquire lock, send multiple heartbeats in sequence, verify each heartbeat extends the expiry from the time of the heartbeat (not from original acquire)

## Phase 3: Release Tests

- [ ] Task 3.1: Release happy path — acquire lock, then DELETE to release, verify 200 response, verify lock no longer appears in `GET /api/locks`
- [ ] Task 3.2: Release clears from list — acquire two locks on different scopes, release one, verify only the released lock is gone from the list while the other remains
- [ ] Task 3.3: Release SSE event — subscribe to SSE before releasing, release a lock, verify `lock_released` event is received with correct scope and holder info

## Phase 4: Conflict Tests

- [ ] Task 4.1: Double acquire conflict — acquire lock on scope, attempt second acquire on same scope with different holder, verify 409 Conflict response with error message indicating lock is held
- [ ] Task 4.2: Lock by different holder — acquire lock as holder-A, attempt heartbeat as holder-B, verify rejection; attempt release as holder-B, verify 403 Forbidden
- [ ] Task 4.3: Wait-for-release scenario — acquire lock with short TTL (3s), attempt second acquire in a polling loop, verify second acquire succeeds after TTL expiry

## Phase 5: TTL and Edge Cases

- [ ] Task 5.1: TTL auto-expiry — acquire lock with 2s TTL, do NOT heartbeat, poll `GET /api/locks` until lock disappears, verify `lock_released` SSE event fires on expiry
- [ ] Task 5.2: Heartbeat after release — acquire and release a lock, then send heartbeat for the released scope, verify 404 response (lock not found), verify no side effects
- [ ] Task 5.3: Invalid TTL — attempt acquire with `ttl_seconds: 0`, `ttl_seconds: -1`, and `ttl_seconds: 999999`, verify appropriate error responses (400 Bad Request)
- [ ] Task 5.4: Wrong holder release — acquire lock as holder-A, attempt DELETE as holder-B, verify 403 Forbidden, verify lock remains active in list with holder-A

## Phase 6: UI Verification

- [ ] Task 6.1: Lock list display — use Playwright to navigate to the dashboard, acquire a lock via API, verify the UI shows the lock with holder name, scope, and remaining TTL countdown
- [ ] Task 6.2: Real-time lock update — with Playwright on the dashboard, acquire a lock via API, verify UI updates without page refresh (SSE-driven); release the lock, verify it disappears from UI
- [ ] Task 6.3: Conflict indicator in UI — acquire a lock, verify UI shows the lock as active; attempt a second acquire via the UI (if applicable) or verify the conflict state is reflected in the lock list display
