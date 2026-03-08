# Implementation Plan: Tour UX Improvements — State Transitions, Demo Data, and Example URL

**Track ID:** tour-ux-improvements_20260309235000Z

## Phase 1: Fix Example URL

- [ ] Task 1.1: Update `tourSteps.ts` step 1 content to reference `https://github.com/kiloforge/example-project`
- [ ] Task 1.2: Update `AddProjectForm.tsx` prefill URL to `https://github.com/kiloforge/example-project`

## Phase 2: Improve Demo Board Data

- [ ] Task 2.1: Update `tour_handler.go` demo board endpoint — return cards spread across multiple columns (backlog, approved, in-progress, done) instead of all in backlog
- [ ] Task 2.2: Add a 4th demo card so each column has representation

## Phase 3: wait-for-drag Escape Hatch

- [ ] Task 3.1: Add "Skip and finish tour" link to `TourOverlay.tsx` for `wait-for-drag` steps
- [ ] Task 3.2: Style the skip link subtly (secondary text, not prominent button)

## Phase 4: Tour Flow Polish

- [ ] Task 4.1: Ensure step content text clearly instructs what user should do at each step
- [ ] Task 4.2: Verify page navigation works — overview → project page transitions happen at correct step boundaries

## Phase 5: Verification

- [ ] Task 5.1: `npm run build` and `make test` pass
- [ ] Task 5.2: Tour walkthrough from start to finish works with correct URL, populated board, and skip option
