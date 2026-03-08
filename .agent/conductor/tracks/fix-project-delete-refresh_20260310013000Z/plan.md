# Implementation Plan: Fix Project Delete — Refresh List and Close Modal

**Track ID:** fix-project-delete-refresh_20260310013000Z

## Phase 1: Fix Delete Flow

- [x] Task 1.1: Convert `removeProject` in `useProjects.ts` to use `useMutation` with `onSuccess` invalidating projects query
- [x] Task 1.2: Ensure `RemoveProjectDialog` properly awaits mutation and closes on success
- [x] Task 1.3: Navigate to overview page if user deletes a project while on that project's page

## Phase 2: Verification

- [x] Task 2.1: Frontend builds without errors (`npm run build`)
- [x] Task 2.2: Delete project → modal closes → project removed from list immediately
- [x] Task 2.3: Delete project from project page → navigates back to overview
