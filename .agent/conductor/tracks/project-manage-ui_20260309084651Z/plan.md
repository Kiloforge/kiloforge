# Implementation Plan: Project Add/Remove Dashboard UI

**Track ID:** project-manage-ui_20260309084651Z

## Phase 1: API Integration Hooks

- [ ] Task 1.1: Add `addProject(remoteUrl, name?)` function to `useProjects` hook — calls `POST /-/api/projects`
- [ ] Task 1.2: Add `removeProject(slug, cleanup?)` function to `useProjects` hook — calls `DELETE /-/api/projects/{slug}`
- [ ] Task 1.3: Add loading/error state management for mutations
- [ ] Task 1.4: Update `api.ts` types — add `AddProjectRequest` interface

## Phase 2: Add Project Form

- [ ] Task 2.1: Create `AddProjectForm` component with Remote URL input and optional Name input
- [ ] Task 2.2: Add client-side URL format validation (SSH, HTTPS patterns)
- [ ] Task 2.3: Add submit handler — call `addProject()`, show loading spinner, handle errors inline
- [ ] Task 2.4: Integrate form into `OverviewPage` above the project table
- [ ] Task 2.5: Update empty state to include the Add Project form instead of CLI-only message
- [ ] Task 2.6: Style with CSS Modules matching existing dashboard design

## Phase 3: Remove Project UI

- [ ] Task 3.1: Add "Remove" button/action to each project row in the Overview table
- [ ] Task 3.2: Create confirmation dialog component — project name, cleanup checkbox, confirm/cancel buttons
- [ ] Task 3.3: Wire confirmation to `removeProject()` hook, show loading state during deletion
- [ ] Task 3.4: Handle success (refresh list) and error (show message) states
- [ ] Task 3.5: Style dialog with CSS Modules

## Phase 4: Verification

- [ ] Task 4.1: Verify frontend builds: `cd frontend && npm run build`
- [ ] Task 4.2: Verify full build: `make build` (embeds frontend into Go binary)
- [ ] Task 4.3: Manual verification: add project via UI, remove project via UI
