# Implementation Plan: Navigation Bar Design Polish and Agent Status Popover

**Track ID:** navbar-design-polish_20260310033000Z

## Phase 1: Nav Bar Spacing and Link Design

### Task 1.1: Fix nav link spacing
- Add `display: flex; gap: 12px; align-items: center` to `<nav>` in `App.module.css`
- Consider pill-style links with padding, border-radius, and subtle background on hover

### Task 1.2: Add "Agents" label to status section
- Add a visual label or separator between the connection status and agent histogram
- Could be a dim text label "Agents:" or a vertical divider + label

### Task 1.3: Improve overall header visual polish
- Review font weights, spacing, and alignment
- Ensure consistent sizing between left and right sections
- Consider subtle hover states for nav links (background highlight instead of just underline)

### Task 1.4: Verify Phase 1
- TypeScript compiles
- Visual inspection of header at different widths

## Phase 2: Agent Status Popover

### Task 2.1: Create AgentStatusPopover component
- Accepts a status string and list of agents
- Renders a positioned dropdown with agent details (name, role, ref)
- Each agent links to `/agents/{id}`
- Styled consistently with existing badge colors

### Task 2.2: Make AgentHistogram chips clickable
- Add onClick handler to each status chip
- Toggle popover open/closed on click
- Close on outside click (useEffect with document click listener) and Escape key

### Task 2.3: Wire popover data
- Filter agents by clicked status
- Pass filtered agents to popover component
- Handle empty state ("No agents")

### Task 2.4: Verify Phase 2
- TypeScript compiles
- Click chip → popover appears with correct agents
- Click outside or Escape → popover closes
- Click agent in popover → navigates to detail page
