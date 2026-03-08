# Implementation Plan: Setup Prerequisite Check — Dashboard UI (Frontend)

**Track ID:** setup-prereq-fe_20260310004001Z

## Phase 1: Hook and Dialog

- [ ] Task 1.1: Create `useSetupPrompt` hook in `frontend/src/hooks/useSetupPrompt.ts` — mirrors useSkillsPrompt pattern
- [ ] Task 1.2: Create `SetupRequiredDialog` component in `frontend/src/components/SetupRequiredDialog.tsx`
- [ ] Task 1.3: "Run Setup" button spawns interactive agent via `POST /api/projects/{slug}/setup` and opens WebSocket terminal
- [ ] Task 1.4: On agent completion, auto-retry the original operation

## Phase 2: Integration

- [ ] Task 2.1: Wire 428 handler in `ProjectPage.tsx` track generation mutation's `onError`
- [ ] Task 2.2: Wire 428 handler in any other agent spawn triggers on the project page
- [ ] Task 2.3: Add setup status banner on project page — poll `GET /api/projects/{slug}/setup-status`, show "Setup Required" banner with action button when incomplete

## Phase 3: Verification

- [ ] Task 3.1: Frontend builds without errors (`npm run build`)
- [ ] Task 3.2: Setup dialog appears when generating tracks on a project without conductor artifacts
- [ ] Task 3.3: After completing setup, track generation proceeds automatically
