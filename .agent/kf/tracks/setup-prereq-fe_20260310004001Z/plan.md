# Implementation Plan: Setup Prerequisite Check — Dashboard UI (Frontend)

**Track ID:** setup-prereq-fe_20260310004001Z

## Phase 1: Hook and Dialog

- [x] Task 1.1: Create `useSetupPrompt` hook in `frontend/src/hooks/useSetupPrompt.ts` — mirrors useSkillsPrompt pattern
- [x] Task 1.2: Create `SetupRequiredDialog` component in `frontend/src/components/SetupRequiredDialog.tsx`
- [x] Task 1.3: "Run Setup" button spawns interactive agent via `POST /api/projects/{slug}/setup` and opens WebSocket terminal
- [x] Task 1.4: On agent completion, auto-retry the original operation

## Phase 2: Integration

- [x] Task 2.1: Wire 428 handler in `ProjectPage.tsx` track generation mutation's `onError`
- [x] Task 2.2: Wire 428 handler in any other agent spawn triggers on the project page
- [x] Task 2.3: Add setup status banner on project page — poll `GET /api/projects/{slug}/setup-status`, show "Setup Required" banner with action button when incomplete

## Phase 3: Verification

- [x] Task 3.1: Frontend builds without errors (`npm run build`)
- [x] Task 3.2: Setup dialog appears when generating tracks on a project without conductor artifacts
- [x] Task 3.3: After completing setup, track generation proceeds automatically
