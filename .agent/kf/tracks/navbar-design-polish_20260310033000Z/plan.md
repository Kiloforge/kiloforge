# Implementation Plan: Navigation Bar Design Polish and Agent Status Popover

**Track ID:** navbar-design-polish_20260310033000Z

## Phase 1: Nav Bar Spacing and Link Design

### Task 1.1: Fix nav link spacing
- [x] Add `display: flex; gap: 8px; align-items: center` to `<nav>` in `App.module.css`
- [x] Pill-style links with padding, border-radius, and subtle background on hover

### Task 1.2: Add "Agents" label to status section
- [x] Added vertical divider + "Agents" label between connection status and histogram

### Task 1.3: Improve overall header visual polish
- [x] Refined title sizing (16px, weight 700, accent color)
- [x] Nav links use pill-style with hover highlight

### Task 1.4: Verify Phase 1
- [x] TypeScript compiles

## Phase 2: Agent Status Popover

### Task 2.1: Create AgentStatusPopover component
- [x] Accepts a status string and list of agents
- [x] Renders a positioned dropdown with agent details (name, role, ref)
- [x] Each agent links to `/agents/{id}`
- [x] Styled consistently with existing badge colors

### Task 2.2: Make AgentHistogram chips clickable
- [x] Add onClick handler to each status chip
- [x] Toggle popover open/closed on click
- [x] Close on outside click (mousedown listener) and Escape key

### Task 2.3: Wire popover data
- [x] Filter agents by clicked status
- [x] Pass filtered agents to popover component
- [x] Handle empty state ("No agents")

### Task 2.4: Verify Phase 2
- [x] TypeScript compiles
