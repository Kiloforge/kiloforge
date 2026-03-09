# Implementation Plan: Branch Diff View with Agent Discuss Button (Frontend)

**Track ID:** branch-diff-view-fe_20260310050001Z

## Phase 1: Dependencies and Role Badge Fix (Foundation)

### Task 1.1: [x] Install react-diff-viewer-continued
- Skipped — built custom lightweight diff renderer instead for React 19 compatibility

### Task 1.2: [x] Add role badge to AgentTerminal modal
- Updated `AgentTerminal` Props to accept `role?: string`
- Added role badge span in the modal header next to agent ID
- Added role badge CSS styles (developer, reviewer, interactive)

### Task 1.3: [x] Add API hooks for diff endpoints
- Created `src/hooks/useDiff.ts` with `useProjectDiff` and `useProjectBranches`
- Added TypeScript types in `src/types/api.ts`: DiffResponse, FileDiff, DiffHunk, DiffLine, DiffStats, BranchInfo
- Added query keys in `src/api/queryKeys.ts`

### Task 1.4: [x] Verify Phase 1
- Build succeeds
- Types and hooks compile correctly

## Phase 2: Diff Viewer Components (Core)

### Task 2.1: [x] Create DiffStats component
- `src/components/diff/DiffStats.tsx` + CSS module
- Shows: "N files changed, +X insertions, -Y deletions"
- Green/red coloring, truncated indicator

### Task 2.2: [x] Create FileList component
- `src/components/diff/FileList.tsx` + CSS module
- Status badges (A/M/D/R) with color coding
- Click handler, active file highlight, per-file +/- counts

### Task 2.3: [x] Create FileDiff component
- `src/components/diff/FileDiff.tsx` + CSS module
- Custom unified diff renderer (not react-diff-viewer-continued)
- Collapsible header, binary file indicator, empty state

### Task 2.4: [x] Create DiffView container
- `src/components/diff/DiffView.tsx` + CSS module
- Composes FileList + DiffStats + FileDiff list
- Loading, empty, error states
- Scroll-to-file coordination via refs

### Task 2.5: [x] Write component tests
- 13 tests covering DiffStats, FileList, FileDiff
- Counts, badges, click handlers, collapse, binary, renamed

### Task 2.6: [x] Verify Phase 2
- All tests passing
- Build clean

## Phase 3: Integration (Polish)

### Task 3.1: [x] Add "View Diff" button to AgentCard
- Shows when agent has `worktree_dir`
- Links to agent detail page with #diff hash

### Task 3.2: [x] Add diff section to AgentDetailPage
- DiffView component above log section
- Hash-based scroll to diff section (#diff)
- Project slug derived from agent's track ref

### Task 3.3: [x] Add "Discuss" button to DiffView
- Header-positioned button in DiffView top bar
- Scrolls to terminal section for interactive agents

### Task 3.4: [x] Verify Phase 3
- All 39 tests passing
- Build clean

---

**Total: 14 tasks across 3 phases — ALL COMPLETE**
