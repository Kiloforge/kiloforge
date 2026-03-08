# Implementation Plan: Agent Display TTL and History Page (Frontend)

**Track ID:** agent-display-ttl-fe_20260310014001Z

## Phase 1: Hook and Data Layer

- [ ] Task 1.1: Update `useAgents` hook to accept `active` parameter, default `true`
- [ ] Task 1.2: Update `queryKeys.agents` to include the `active` param for proper cache separation
- [ ] Task 1.3: Update `AgentHistogram` to only count active agents (running, waiting)

## Phase 2: Dashboard Update

- [ ] Task 2.1: Dashboard `AgentGrid` uses `useAgents(true)` — only active + recently finished
- [ ] Task 2.2: Add "View all" link next to Agents section header, navigates to `/agents`
- [ ] Task 2.3: Optional: add "finished X ago" indicator on recently finished agent cards

## Phase 3: Agent History Page

- [ ] Task 3.1: Create `AgentHistoryPage` component with table layout
- [ ] Task 3.2: Fetch all agents via `useAgents(false)`
- [ ] Task 3.3: Add status filter dropdown (All, Running, Completed, Failed, etc.)
- [ ] Task 3.4: Clickable rows navigate to `/agents/:id` detail page
- [ ] Task 3.5: Add `/agents` route to `App.tsx`
- [ ] Task 3.6: Add "Agents" link to navigation/sidebar if one exists

## Phase 4: Verification

- [ ] Task 4.1: Frontend builds without errors (`npm run build`)
- [ ] Task 4.2: Dashboard only shows active + recently finished agents
- [ ] Task 4.3: Status bar only counts running/waiting agents
- [ ] Task 4.4: History page shows all agents with filtering
