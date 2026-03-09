# Implementation Plan: Interactive Agent Terminal in Dashboard (Frontend)

**Track ID:** interactive-agent-fe_20260309150001Z

## Phase 1: WebSocket Hook

- [x] Task 1.1: Create `frontend/src/hooks/useAgentWebSocket.ts` — connect to `/ws/agent/{id}`, handle messages, reconnection, send input
- [x] Task 1.2: Define message types in `types/api.ts` (AgentWSMessage, input/output/status/error)

## Phase 2: Terminal Component

- [x] Task 2.1: Create `frontend/src/components/AgentTerminal.tsx` — chat-style message display with auto-scroll, markdown rendering for agent output, plain text for user input
- [x] Task 2.2: Create input area component — text input with Send button and Enter-to-send
- [x] Task 2.3: Add connection status indicator (green dot connected, yellow reconnecting, red disconnected)
- [x] Task 2.4: Style agent vs user messages differently (alignment, color, icon)

## Phase 3: Integration

- [x] Task 3.1: Add "Start Interactive Agent" action — button that calls `POST /api/agents/interactive`, opens terminal on success
- [x] Task 3.2: Add "Attach" button to agent cards for interactive agents — opens terminal connected to existing agent
- [x] Task 3.3: Create interactive agent page/modal that hosts the terminal component
- [x] Task 3.4: Add route for interactive terminal view (e.g., `/agents/{id}/terminal`)

## Phase 4: Verification

- [x] Task 4.1: Verify `npm run build` succeeds
- [x] Task 4.2: Verify WebSocket connects and displays agent output
- [x] Task 4.3: Verify user input is sent and agent responds
- [x] Task 4.4: Verify reconnection replays buffered output
