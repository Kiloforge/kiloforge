# Implementation Plan: E2E Tests: Track Management — List, Detail, Generate, and Delete

**Track ID:** e2e-track-management_20260309194833Z

## Phase 1: Track List Tests

- [ ] Task 1.1: Create `frontend/e2e/track-list.spec.ts` — seed 5 tracks with various statuses (pending, approved, in_progress, in_review, done) via `seedTestData()`, navigate to track list page, verify all tracks displayed with correct title, status badge, project name, and timestamps
- [ ] Task 1.2: Test empty track list — start with no seeded tracks, navigate to track list, verify empty state message displayed (e.g., "No tracks yet"), verify "Generate Tracks" call-to-action is visible
- [ ] Task 1.3: Test track filtering and sorting — seed 5 tracks, test status filter dropdown (select "pending", verify only pending tracks shown), test sort by date (ascending/descending), test sort by title (alphabetical)

## Phase 2: Track Detail Tests

- [ ] Task 2.1: Create `frontend/e2e/track-detail.spec.ts` — seed a track with full spec/plan/metadata, click track in list to navigate to detail page, verify spec tab shows rendered markdown content
- [ ] Task 2.2: Test track detail tab navigation — verify plan tab shows task list with checkboxes, verify metadata tab shows track metadata (type, status, created date, phase/task counts), verify tab switching preserves scroll position
- [ ] Task 2.3: Test nonexistent track navigation — navigate to `/tracks/nonexistent-id`, verify 404 page or "Track not found" message displayed, verify back navigation returns to track list

## Phase 3: Track Generation Tests

- [ ] Task 3.1: Create `frontend/e2e/track-generate.spec.ts` — seed a project, open track generation dialog, fill in prompt/description, submit, verify mock agent is spawned (agent appears in agent list or status indicator shows "running")
- [ ] Task 3.2: Test real-time stream display — trigger generation, verify stream output panel appears, verify text from mock agent `content_block_delta` events renders progressively in the output area, verify init event shows session info
- [ ] Task 3.3: Test generation completion — wait for mock agent `result` event, verify generation completes successfully, verify new track appears in track list, verify track has correct metadata (usage stats from result event)
- [ ] Task 3.4: Test generation failure handling — configure mock agent with `MOCK_AGENT_EXIT_CODE=1` and `MOCK_AGENT_FAIL_AFTER=2`, trigger generation, verify error state displayed in UI, verify error toast notification, verify no partial/corrupt track created in list

## Phase 4: Track Deletion Tests

- [ ] Task 4.1: Create `frontend/e2e/track-delete.spec.ts` — seed a track, navigate to track detail or list, click delete button, verify confirmation dialog appears with track title, confirm deletion, verify track removed from list
- [ ] Task 4.2: Test cancel delete — seed a track, click delete button, verify confirmation dialog, click cancel, verify track still exists in list and detail page still accessible
- [ ] Task 4.3: Test delete nonexistent track — call `DELETE /api/tracks/nonexistent-id` directly via API, verify 404 response, verify UI handles gracefully if track was already deleted (e.g., another tab deleted it)

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Test concurrent track generation — trigger two track generations simultaneously for the same project, verify both succeed or one queues gracefully, verify no duplicate tracks or data corruption
- [ ] Task 5.2: Test long content handling — seed a track with very long title (200+ characters) and very long spec content (10,000+ characters), verify list truncates title appropriately, verify detail page renders full content with scrolling
- [ ] Task 5.3: Test API error scenarios — mock 500 error on track list endpoint (verify error state with retry button), mock timeout on track generation (verify timeout message after reasonable wait), mock 403 on track delete (verify permission error toast)
