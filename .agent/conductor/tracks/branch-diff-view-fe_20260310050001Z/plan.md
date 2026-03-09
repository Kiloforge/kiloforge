# Implementation Plan: Branch Diff View with Agent Discuss Button (Frontend)

**Track ID:** branch-diff-view-fe_20260310050001Z

## Phase 1: Dependencies and Role Badge Fix (Foundation)

### Task 1.1: Install react-diff-viewer-continued
- `npm install react-diff-viewer-continued`
- Verify it works with React 19 and Vite build

### Task 1.2: Add role badge to AgentTerminal modal
- Update `AgentTerminal` Props to accept `role?: string`
- Add role badge span in the modal header next to agent ID (reuse AgentCard's badge styling)
- Update all AgentTerminal call sites to pass the role prop
- Update AgentTerminal CSS module with role badge styles (reuse from AgentCard.module.css)

### Task 1.3: Add API hooks for diff endpoints
- Create `src/hooks/useDiff.ts` with:
  - `useProjectDiff(slug: string, branch: string)` — TanStack useQuery for `GET /api/projects/{slug}/diff?branch={branch}`
  - `useProjectBranches(slug: string)` — TanStack useQuery for `GET /api/projects/{slug}/branches`
- Add TypeScript types in `src/types/api.ts`: `DiffResponse`, `FileDiff`, `Hunk`, `DiffLine`, `DiffStats`, `BranchInfo`

### Task 1.4: Verify Phase 1
- Build succeeds with new dependency
- Role badge renders in terminal modal
- Hooks compile correctly (endpoint may 404 until BE track is done — that's fine)

## Phase 2: Diff Viewer Components (Core)

### Task 2.1: Create DiffStats component
- `src/components/diff/DiffStats.tsx` + CSS module
- Shows: "N files changed, +X insertions, -Y deletions"
- Green/red coloring for insertions/deletions
- Compact inline bar visualization

### Task 2.2: Create FileList component
- `src/components/diff/FileList.tsx` + CSS module
- Vertical list of changed files with:
  - Status badge (A = added/green, M = modified/yellow, D = deleted/red, R = renamed/blue)
  - File path (truncated with tooltip for long paths)
  - Per-file +/- counts
- Click handler to scroll to file's diff section
- Highlight currently-visible file

### Task 2.3: Create FileDiff component
- `src/components/diff/FileDiff.tsx` + CSS module
- Renders a single file's diff using react-diff-viewer-continued
- Collapsible header with file path and status
- Dark theme matching dashboard design tokens
- Binary file indicator (no diff content, just a label)
- "No changes" state for empty hunks

### Task 2.4: Create DiffView container
- `src/components/diff/DiffView.tsx` + CSS module
- Main container that composes FileList + DiffStats + FileDiff list
- Fetches diff data via `useProjectDiff` hook
- Loading skeleton state
- Empty state: "No changes on this branch"
- Error state: branch not found, API error
- Scroll-to-file coordination between FileList clicks and FileDiff sections

### Task 2.5: Write component tests
- Test DiffStats renders correct counts
- Test FileList renders files with correct status badges
- Test DiffView loading/empty/error states
- Test FileList click triggers scroll

### Task 2.6: Verify Phase 2
- All diff components render correctly with mock data
- Tests passing
- Build clean

## Phase 3: Integration (Polish)

### Task 3.1: Add "View Diff" button to AgentCard
- Show button only when agent has `worktree_dir` and status is running/waiting/halted/completed
- Button opens diff view (either navigates to agent detail with diff tab, or opens a modal)
- Determine branch from agent's track ID or worktree branch

### Task 3.2: Add diff section to AgentDetailPage
- Add DiffView component below existing terminal section
- Pass project slug and branch from agent's worktree info
- Add "Discuss" button that opens/focuses the agent terminal section

### Task 3.3: Add "Discuss" button to DiffView
- Floating or header-positioned button
- Opens AgentTerminal for the agent associated with the branch
- Only shown when an agent is actively working on the branch

### Task 3.4: Verify Phase 3
- End-to-end flow: AgentCard → View Diff → see changes → Discuss → terminal opens
- All tests passing
- Build clean
- Manual verification with running agents

---

**Total: 14 tasks across 3 phases**
