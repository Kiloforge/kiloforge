# Implementation Plan: Fix Tour API Request Body Mismatch

**Track ID:** fix-tour-api-mismatch_20260309223000Z

## Phase 1: Fix

- [x] Task 1.1: Update `useTour.ts` — change mutation to send `{ action, step? }` instead of `Partial<TourState>`
- [x] Task 1.2: Update `startTour()` → `{ action: "accept" }`
- [x] Task 1.3: Update `advanceStep(step)` → `{ action: "advance", step }`
- [x] Task 1.4: Update `dismissTour()` → `{ action: "dismiss" }`
- [x] Task 1.5: Update `completeTour()` → `{ action: "complete" }`

## Phase 2: Verification

- [x] Task 2.1: `npm run build` succeeds
- [x] Task 2.2: Rebuild dist and commit
- [x] Task 2.3: Manual test — full tour walkthrough works without 400 errors
