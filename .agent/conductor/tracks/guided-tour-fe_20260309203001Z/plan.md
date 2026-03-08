# Implementation Plan: Guided Tour Overlay with Simulated Onboarding Flow (Frontend)

**Track ID:** guided-tour-fe_20260309203001Z

## Phase 1: Tour Infrastructure

- [ ] Task 1.1: Install lightweight tour/tooltip library if needed (or build custom — assess trade-offs)
- [ ] Task 1.2: Create `frontend/src/hooks/useTour.ts` — TanStack Query hook for tour state CRUD
- [ ] Task 1.3: Add `tour` to `queryKeys.ts`
- [ ] Task 1.4: Create `frontend/src/components/tour/TourProvider.tsx` — context provider wrapping App, manages step progression
- [ ] Task 1.5: Create `frontend/src/components/tour/TourOverlay.tsx` — backdrop + spotlight + tooltip renderer

## Phase 2: Tour Step Definitions and Welcome

- [ ] Task 2.1: Define `TOUR_STEPS` array with all 7 steps (target selectors, content, actions)
- [ ] Task 2.2: Create welcome modal (Step 1) — "Start Tour" / "Skip" buttons, persists dismissal via `PUT /api/tour`
- [ ] Task 2.3: Auto-launch welcome modal when tour state is `"pending"` on first page load

## Phase 3: Add Project and Navigation Steps

- [ ] Task 3.1: Add `data-tour` attributes to `AddProjectForm`, project cards, and other target elements
- [ ] Task 3.2: Implement Step 2 — auto-expand Add Project form, prefill `remote_url` with example repo URL
- [ ] Task 3.3: Implement Step 3 — after project add succeeds, highlight new project card with "Click to open" tooltip
- [ ] Task 3.4: Implement Step 4 — on ProjectPage mount during tour, show setup explanation tooltip

## Phase 4: Simulated Track Generation and Board

- [ ] Task 4.1: Implement Step 5 — highlight "Generate Tracks" button, prefill prompt textarea, on submit show fake loading animation then inject demo board data from `GET /api/tour/demo-board`
- [ ] Task 4.2: Implement Step 6 — board column annotation overlay explaining each column's role
- [ ] Task 4.3: Implement Step 7 — highlight specific backlog card, detect drag to "approved", fire completion
- [ ] Task 4.4: Tour completion screen — congratulatory message, "Got it" button that marks tour complete

## Phase 5: Polish and Verification

- [ ] Task 5.1: Add "Restart Tour" option in header or settings area
- [ ] Task 5.2: Ensure tour overlay doesn't break existing functionality when dismissed mid-step
- [ ] Task 5.3: `npm run build` succeeds
- [ ] Task 5.4: Manual walkthrough — complete full tour end-to-end, verify all steps, verify dismissal persists
