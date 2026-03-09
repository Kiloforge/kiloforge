# Specification: Fix Unused Variable Errors in TraceTimeline

**Track ID:** fix-trace-timeline-unused-vars_20260308220100Z
**Type:** Bug
**Created:** 2026-03-08T22:01:00Z
**Status:** Draft

## Summary

Fix TypeScript compilation errors in `frontend/src/components/TraceTimeline.tsx` caused by unused destructured variables `minTime` and `totalDuration`.

## Context

The frontend build fails with TS6133 errors:

```
src/components/TraceTimeline.tsx:11:17 - error TS6133: 'minTime' is declared but its value is never read.
src/components/TraceTimeline.tsx:11:26 - error TS6133: 'totalDuration' is declared but its value is never read.
```

Line 11 destructures `{ rows, minTime, totalDuration }` from a `useMemo` call, but only `rows` is used in the component's render output. The `minTime` and `totalDuration` values are computed inside the memo but never referenced outside it.

## Codebase Analysis

- **`frontend/src/components/TraceTimeline.tsx`** — The component destructures three values but only uses `rows`. The `minTime` and `totalDuration` are computed for potential future use but are currently unused.

## Acceptance Criteria

- [ ] Frontend build (`make build` or `npm run build` in frontend/) compiles without errors
- [ ] No behavior change — the component renders identically
- [ ] The computed values inside useMemo remain available if needed later (don't remove the computation, just fix the destructuring)

## Dependencies

None

## Out of Scope

- Refactoring TraceTimeline beyond fixing the build error
- Adding usage of minTime/totalDuration (separate track if needed)

## Technical Notes

Fix: change destructuring to only extract `rows`, or prefix unused vars with `_`:
```typescript
const { rows, minTime: _minTime, totalDuration: _totalDuration } = useMemo(...)
```

Or simpler — just extract `rows`:
```typescript
const { rows } = useMemo(...)
```

The simpler approach is preferred since the unused values have no side effects.
