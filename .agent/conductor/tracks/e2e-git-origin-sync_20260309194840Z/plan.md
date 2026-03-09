# Implementation Plan: E2E Tests — Git Origin Sync

**Track ID:** e2e-git-origin-sync_20260309194840Z

## Phase 1: Sync Status Display Tests

- [ ] Task 1.1: Synced state — seed a project where local and origin are at the same commit, navigate to project page, verify sync status shows "synced" with green indicator, verify both push and pull buttons are disabled
- [ ] Task 1.2: Ahead state — seed a project with 2 local commits not pushed to origin, navigate to project page, verify sync status shows "ahead" with commit count (2), verify push button is enabled and pull button is disabled
- [ ] Task 1.3: Behind state — seed a project where origin has 3 commits not yet pulled, navigate to project page, verify sync status shows "behind" with commit count (3), verify pull button is enabled and push button is disabled
- [ ] Task 1.4: Diverged state — seed a project where local has 1 unpushed commit and origin has 2 unpulled commits, navigate to project page, verify status shows "diverged" with both counts, verify both push and pull buttons are enabled

## Phase 2: Push Tests

- [ ] Task 2.1: Push happy path — seed project in "ahead" state, click push button via Playwright, verify POST to `/api/projects/{slug}/push` succeeds, verify success feedback displayed (toast or inline message)
- [ ] Task 2.2: Push feedback — after successful push, verify sync status updates to "synced", verify push button becomes disabled, verify ahead count resets to 0
- [ ] Task 2.3: Push disables when synced — seed project in "synced" state, verify push button is disabled and not clickable, verify no POST request is sent when attempting to interact with disabled button

## Phase 3: Pull Tests

- [ ] Task 3.1: Pull happy path — seed project in "behind" state, click pull button via Playwright, verify POST to `/api/projects/{slug}/pull` succeeds, verify success feedback displayed
- [ ] Task 3.2: Pull feedback — after successful pull, verify sync status updates to "synced", verify pull button becomes disabled, verify behind count resets to 0
- [ ] Task 3.3: Pull disables when synced — seed project in "synced" state, verify pull button is disabled and not clickable

## Phase 4: Sync Panel Tests

- [ ] Task 4.1: Panel updates after operations — perform push on an "ahead" project, verify panel updates from "ahead" to "synced" without page reload; perform pull on a "behind" project, verify panel updates similarly
- [ ] Task 4.2: Diverged state UI — seed diverged project, verify UI shows both ahead and behind counts, verify both push and pull buttons are present with appropriate labels, verify user can choose which operation to perform
- [ ] Task 4.3: Refresh behavior — navigate to project page, change sync state via API (add commit to origin), trigger manual refresh or wait for polling interval, verify status updates from "synced" to "behind"

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Concurrent operations — seed "diverged" project, initiate push, immediately attempt pull (or vice versa), verify the server handles gracefully (queues or rejects second operation), verify no UI corruption
- [ ] Task 5.2: Unreachable remote — seed project with a non-routable origin URL (e.g., `git://192.0.2.1/repo.git`), attempt push, verify error feedback is displayed with meaningful message (e.g., "Remote unreachable"), verify sync status remains unchanged
- [ ] Task 5.3: Merge conflict handling — seed diverged project where the same file is modified in both local and origin, attempt pull, verify error feedback indicates merge conflict, verify sync status still shows "diverged"
