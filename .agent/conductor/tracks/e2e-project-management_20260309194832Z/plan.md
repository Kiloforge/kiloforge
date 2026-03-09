# Implementation Plan: E2E Tests: Project Management — Add, Remove, Setup, and Sync Status

**Track ID:** e2e-project-management_20260309194832Z

## Phase 1: Add Project Tests

- [ ] Task 1.1: Create `frontend/e2e/project-add.spec.ts` — test add project with valid HTTPS URL (`https://github.com/user/repo.git`), verify project appears in project list, verify project detail page accessible
- [ ] Task 1.2: Test add project with valid SSH URL — submit `git@github.com:user/repo.git`, verify project created, verify SSH URL displayed correctly in project detail
- [ ] Task 1.3: Test add project form validation — submit empty URL (verify validation error), submit invalid URL like `not-a-url` (verify validation error), verify submit button disabled while invalid
- [ ] Task 1.4: Test duplicate project handling — add a project, attempt to add same URL again, verify error toast appears with "already exists" message, verify original project unchanged

## Phase 2: Remove Project Tests

- [ ] Task 2.1: Create `frontend/e2e/project-remove.spec.ts` — test remove project happy path: seed a project, click delete button, verify confirmation dialog appears, confirm deletion, verify project removed from list
- [ ] Task 2.2: Test remove project with cleanup — seed a project, click delete, check "clean up local data" checkbox in confirmation dialog, confirm, verify project and local data removed
- [ ] Task 2.3: Test cancel remove — seed a project, click delete, verify confirmation dialog, click cancel, verify project still in list and unchanged

## Phase 3: Setup Status Tests

- [ ] Task 3.1: Create `frontend/e2e/project-status.spec.ts` — test project with complete setup: seed fully configured project, verify setup status shows "complete" on project detail page
- [ ] Task 3.2: Test project with incomplete setup — seed project missing configuration steps, verify setup status shows "incomplete" with list of missing steps
- [ ] Task 3.3: Test setup status transitions — start with incomplete project, complete a setup step via API, verify status updates in UI (poll or SSE-driven)

## Phase 4: Sync Status Tests

- [ ] Task 4.1: Test sync status display — seed projects with different sync states (synced, ahead, behind, diverged), navigate to project list, verify each project shows correct sync badge color and label
- [ ] Task 4.2: Test sync status on project detail page — navigate to individual project, verify detailed sync info including commit counts (e.g., "2 commits ahead")
- [ ] Task 4.3: Test sync status refresh — trigger a sync status refresh (button click or page reload), verify status updates if underlying state changed

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Test unreachable URL handling — attempt to add project with URL `https://example.com/nonexistent-repo.git`, verify API returns error, verify error toast shows user-friendly message, verify no phantom project created
- [ ] Task 5.2: Test API error toast display — mock various API error responses (400, 404, 500), verify each shows an appropriate toast notification with descriptive message, verify toasts auto-dismiss
- [ ] Task 5.3: Test concurrent operations — add two projects simultaneously, verify both succeed or one fails gracefully; delete a project while another add is in progress, verify no corruption
