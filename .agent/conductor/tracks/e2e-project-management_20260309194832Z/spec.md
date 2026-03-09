# Specification: E2E Tests: Project Management — Add, Remove, Setup, and Sync Status

**Track ID:** e2e-project-management_20260309194832Z
**Type:** Chore
**Created:** 2026-03-09T19:48:32Z
**Status:** Draft

## Summary

Comprehensive E2E tests for the project lifecycle: adding projects via remote URL, removing projects with cleanup, project setup wizard, and Git sync status display, covering happy path, edge cases, and expected failures.

## Context

Project management is a core workflow in Kiloforge. Users add projects by providing a Git remote URL (HTTPS or SSH), Kiloforge clones the repo locally via Gitea, and the project becomes available for track generation and agent management. The project lifecycle includes:

1. **Add project** — provide remote URL, Kiloforge creates a Gitea mirror
2. **Remove project** — delete project with optional cleanup of local data
3. **Setup status** — check whether a project is fully configured (repo cloned, skills installed, etc.)
4. **Sync status** — display whether the local repo is synced, ahead, behind, or diverged from origin

The REST API provides `POST /api/projects`, `DELETE /api/projects/{id}`, `GET /api/projects/{id}/status`, and `GET /api/projects/{id}/sync` endpoints. The frontend has an AddProjectDialog, project list view, project detail page, and sync status badges.

## Codebase Analysis

### Existing patterns

- **Project API** — `POST /api/projects` accepts `{url: string}`, validates URL format, creates Gitea repo mirror
- **Project list** — `GET /api/projects` returns all projects with status
- **Project detail** — `GET /api/projects/{id}` returns project info including sync state
- **Delete API** — `DELETE /api/projects/{id}` with optional `?cleanup=true` query param
- **Frontend components** — `ProjectList`, `AddProjectDialog`, `ProjectDetail`, `SyncStatusBadge`
- **URL validation** — backend validates HTTPS (`https://`) and SSH (`git@`) URL formats

### Mock agent relevance

Project add/remove operations do not directly spawn agents, but setup status checks may verify agent availability. The mock agent is needed for the E2E server infrastructure, not for project-specific operations.

## Acceptance Criteria

- [ ] Add project with valid HTTPS URL — project appears in list
- [ ] Add project with valid SSH URL — project appears in list
- [ ] Add project form validates empty and invalid URLs
- [ ] Remove project with confirmation dialog — project disappears
- [ ] Remove project with cleanup option
- [ ] Project setup status check returns correct state
- [ ] Sync status displays: synced, ahead, behind, diverged
- [ ] Project detail page shows correct info
- [ ] Edge cases: duplicate project URL, project with special characters in slug
- [ ] Failure cases: add project with unreachable URL, remove nonexistent project, API errors show toast

## Dependencies

- `e2e-infra-mock-agent_20260309194830Z` — E2E test infrastructure and mock agent binary

## Blockers

None.

## Conflict Risk

- LOW — adds test files only, no production code changes.

## Out of Scope

- Git push/pull operations (covered by `e2e-git-origin-sync_20260309194840Z`)
- Kanban board interactions (covered by `e2e-kanban-board_20260309194836Z`)
- Agent spawning within projects
- SSH key management during project add

## Technical Notes

### Test file structure

```
frontend/e2e/
  project-add.spec.ts       — add project tests (HTTPS, SSH, validation)
  project-remove.spec.ts    — remove project tests (confirm, cleanup, cancel)
  project-status.spec.ts    — setup status and sync status tests
```

### Add project test scenarios

| Scenario | URL | Expected |
|---|---|---|
| Valid HTTPS | `https://github.com/user/repo.git` | Project created, appears in list |
| Valid SSH | `git@github.com:user/repo.git` | Project created, appears in list |
| Empty URL | `` | Validation error shown |
| Invalid URL | `not-a-url` | Validation error shown |
| Duplicate URL | Same URL as existing project | Error toast: project already exists |
| Unreachable URL | `https://example.com/nonexistent.git` | Error toast: could not reach remote |

### Remove project test scenarios

- Remove with confirmation: click delete, confirm in dialog, project removed from list
- Remove with cleanup: check cleanup checkbox, confirm, verify local data cleaned
- Cancel remove: click delete, cancel in dialog, project still in list
- Remove nonexistent: `DELETE /api/projects/999` returns 404, UI handles gracefully

### Sync status display

The sync status badge shows one of:
- **Synced** (green) — local matches remote
- **Ahead** (blue) — local has unpushed commits
- **Behind** (yellow) — remote has new commits
- **Diverged** (red) — both local and remote have new commits

Test by seeding projects with different sync states and verifying badge colors/labels.

### Error toast verification

All API errors should display a toast notification. Tests should verify:
- Toast appears with correct error message
- Toast auto-dismisses after timeout
- Multiple errors show multiple toasts (not overwriting)

### Developer agent instructions

Use the Playwright MCP skill to verify all UI interactions. Pay special attention to:
- Dialog open/close animations
- Form validation feedback timing
- Toast notification positioning and auto-dismiss
- Project list updates after add/remove (may need to wait for SSE update or manual refresh)

---

_Generated by conductor-track-generator for E2E project management tests_
