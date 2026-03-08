# Specification: Guided Tour Overlay with Simulated Onboarding Flow (Frontend)

**Track ID:** guided-tour-fe_20260309203001Z
**Type:** Feature
**Created:** 2026-03-09T20:30:01Z
**Status:** Draft

## Summary

Build a step-by-step guided tour overlay that walks new users through the entire Kiloforge workflow using simulated data. The tour highlights UI elements with tooltips, prefills forms with example data, simulates track generation (no real agents), explains the kanban board, and prompts the user to drag a card from backlog. Tour state persists via the backend API so dismissal survives across sessions.

## Context

New users see an empty dashboard with no projects and no obvious next step. The guided tour provides a hands-on walkthrough of the complete workflow: add a project → view the project → generate tracks → understand the board → move a card. All heavy operations (agent spawning, real project setup) are simulated so the tour works instantly without prerequisites.

## Codebase Analysis

### Existing UI patterns

- **ConsentDialog** (`frontend/src/components/ConsentDialog.tsx`) — Modal overlay with accept/deny actions. Tour welcome dialog can follow the same pattern.
- **React Router** — `BrowserRouter` with 4 routes. Tour needs to navigate between OverviewPage and ProjectPage.
- **TanStack Query** — All data fetching uses `useQuery`/`useMutation`. Tour data comes from `/api/tour/demo-board` via a dedicated hook.
- **KanbanBoard** (`frontend/src/components/KanbanBoard.tsx`) — Drag-and-drop cards between columns. Tour highlights a specific card and detects when user drags it.
- **AddProjectForm** (`frontend/src/components/AddProjectForm.tsx`) — Collapsible form with `remote_url` field. Tour prefills this field.
- **CSS** — Tailwind-style utility classes used throughout. Tour overlay uses z-index layering with backdrop dimming.

### Tour step flow

| Step | Page | Action | Simulated? |
|------|------|--------|-----------|
| 1. Welcome | OverviewPage | Accept/dismiss tour | No (real UI) |
| 2. Add Project | OverviewPage | Highlight form, prefill example URL | Partially (real form, example URL) |
| 3. Open Project | OverviewPage | After add, highlight project card → click | No (real navigation) |
| 4. Setup Notice | ProjectPage | Explain `kf add` handles repo setup | Yes (informational tooltip) |
| 5. Generate Tracks | ProjectPage | Highlight "Generate Tracks", prefill prompt | Yes (simulated — fake cards appear after delay) |
| 6. Board Explanation | ProjectPage | Annotate kanban columns | Yes (overlay annotations) |
| 7. Move Card | ProjectPage | Instruct drag from backlog → approved | Partially (real drag on demo data) |

### Not simulated

- Adding a project (Step 2) — this actually calls `POST /api/projects` with the prefilled URL. The example repo should be a real public repo so the add succeeds. If the URL is invalid or unreachable, the tour gracefully skips to step 4 with a note.
- Navigation (Step 3) — real React Router navigation to `/projects/{slug}`.
- Card drag (Step 7) — real drag interaction on demo board data.

### Simulated

- Track generation (Step 5) — instead of spawning an agent, fetches demo cards from `GET /api/tour/demo-board` and injects them into the board view after a 2-second fake "generating..." animation.
- Board data (Steps 5-7) — demo board data replaces real board data during tour mode on the ProjectPage.

## Acceptance Criteria

- [ ] Tour launches automatically on first visit (tour state = `"pending"`)
- [ ] Welcome modal with "Start Tour" and "Skip" buttons
- [ ] "Skip" permanently dismisses tour (persisted via `PUT /api/tour`)
- [ ] Tour tooltip/highlight overlay points to the current UI element with descriptive text
- [ ] Step 2: Add Project form opens automatically, `remote_url` prefilled with example public repo
- [ ] Step 3: After project added, project card highlighted with "Click to open" prompt
- [ ] Step 4: On ProjectPage, tooltip explains project setup is handled by `kf add`
- [ ] Step 5: "Generate Tracks" highlighted, prompt textarea prefilled. On "Generate", fake loading animation plays, then demo cards appear on board (no real agent)
- [ ] Step 6: Board columns annotated with role descriptions (backlog = new, approved = ready for dev, etc.)
- [ ] Step 7: Specific backlog card highlighted with "Drag this to Approved" instruction. Tour completes when card is moved.
- [ ] Tour completion celebration (confetti or simple "You're ready!" message)
- [ ] Tour state synced to backend on every step transition
- [ ] Tour can be restarted from a settings/help menu
- [ ] Tour overlay does not break existing functionality when dismissed
- [ ] `npm run build` succeeds
- [ ] No regressions to existing pages

## Dependencies

- `guided-tour-be_20260309203000Z` — Backend tour state API and demo board endpoint

## Blockers

None.

## Conflict Risk

- MEDIUM against `tanstack-query-migration` — both touch hooks and data fetching patterns. This track should land AFTER TanStack migration is complete. (Already complete on main.)
- LOW against `agent-list-monitoring-ui` — different page focus.

## Out of Scope

- Real agent spawning during tour
- Real `kf setup` command (future track)
- Creating a demo GitHub repository
- Tour for CLI commands (this is dashboard-only)
- Mobile responsive tour layout
- Localization/i18n

## Technical Notes

### Tour provider

```tsx
// frontend/src/components/tour/TourProvider.tsx
interface TourStep {
  id: string;
  target: string;        // CSS selector for highlight target
  title: string;
  content: string;
  page?: string;          // route to navigate to
  action?: 'prefill' | 'simulate-generate' | 'wait-for-drag';
  actionData?: Record<string, string>;
}

const TOUR_STEPS: TourStep[] = [
  {
    id: 'welcome',
    target: 'body',
    title: 'Welcome to Kiloforge!',
    content: 'Let\'s walk through how to set up your first project and start automating development.',
  },
  {
    id: 'add-project',
    target: '[data-tour="add-project-form"]',
    title: 'Add a Project',
    content: 'Start by adding a Git repository. We\'ve prefilled an example for you.',
    action: 'prefill',
    actionData: { remote_url: 'https://github.com/kiloforge/demo-project.git' },
  },
  // ... remaining steps
];
```

### Tour hook

```tsx
// frontend/src/hooks/useTour.ts
function useTour() {
  const { data: tourState } = useQuery({
    queryKey: queryKeys.tour,
    queryFn: () => fetch('/api/tour').then(r => r.json()),
  });

  const updateTour = useMutation({
    mutationFn: (state) => fetch('/api/tour', { method: 'PUT', body: JSON.stringify(state) }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.tour }),
  });

  return { tourState, startTour, advanceStep, dismissTour, completeTour };
}
```

### Highlight overlay

Uses a combination of:
1. **Backdrop** — full-screen semi-transparent overlay (z-index: 1000)
2. **Spotlight** — CSS clip-path or box-shadow cutout around the target element
3. **Tooltip** — positioned relative to target with arrow, contains step title/content/next button

### Demo board injection

During tour Steps 5-7, the ProjectPage board component checks `tourState.status === 'active'` and replaces real board data with demo data from `/api/tour/demo-board`. When tour completes or is dismissed, real board data resumes.

### Data-tour attributes

Add `data-tour="..."` attributes to key elements for reliable targeting:
- `data-tour="add-project-form"` on AddProjectForm
- `data-tour="project-card-{slug}"` on project cards
- `data-tour="generate-tracks"` on generate button
- `data-tour="kanban-board"` on board container
- `data-tour="board-column-{name}"` on each column
- `data-tour="board-card-{id}"` on each card

---

_Generated by conductor-track-generator from prompt: "guided tour mode on startup with simulated demo flow"_
