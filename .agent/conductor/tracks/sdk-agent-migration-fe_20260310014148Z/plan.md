# Implementation Plan: Structured Agent Terminal Display (Frontend)

**Track ID:** sdk-agent-migration-fe_20260310014148Z

## Phase 1: WebSocket Hook Update

### Task 1.1: Extend WSMessage type
- Add new fields to `WSMessage`: `turnId`, `toolName`, `toolId`, `toolInput`, `thinking`, `costUsd`, `usage`, `subtype`, `data`
- Add new message type literals to the type union

### Task 1.2: Update message handler in useAgentWebSocket
- Add cases for `turn_start`, `text`, `tool_use`, `thinking`, `turn_end`, `system` in the message handler switch
- Map legacy `output` messages to `text` type for backward compatibility
- Parse structured fields from server JSON into `WSMessage`

### Task 1.3: Verify hook handles all message types
- Manual test with mock WS messages
- Ensure backward compat: old `output` messages still render

## Phase 2: Terminal Component Refactor

### Task 2.1: Extract shared terminal components
- Create `frontend/src/components/terminal/` directory
- Extract `MessageBubble` from `AgentTerminal.tsx` into shared component
- Extract `formatCode` utility into shared module
- Update `AgentTerminal.tsx` and `AgentDetailPage.tsx` to import shared components

### Task 2.2: Create ToolUseBubble component
- Display tool name as a styled badge/chip
- Show tool ID in muted text
- Collapsible JSON input display (collapsed by default)
- Use monospace font for JSON, syntax-style colors

### Task 2.3: Create ThinkingBubble component
- Distinct visual style: muted background, italic text
- Collapsible (collapsed by default for long thinking blocks)
- "Thinking..." label with brain/thought icon or indicator

### Task 2.4: Create TurnSeparator component
- Horizontal rule with turn number centered
- Subtle styling that doesn't dominate the terminal

### Task 2.5: Create TurnEndSummary component
- Inline summary below turn separator
- Show: tokens used (input/output), cost in USD
- Muted/small text, right-aligned or centered

### Task 2.6: Create SystemBubble component
- Severity-based styling: warning (yellow), error (red), info (blue)
- Subtype label + message text
- Appropriate icon per severity

## Phase 3: Integration

### Task 3.1: Update AgentTerminal message rendering
- Replace single `MessageBubble` switch with component dispatch based on message type
- `text` / `output` → TextBubble (existing)
- `tool_use` → ToolUseBubble
- `thinking` → ThinkingBubble
- `turn_start` → TurnSeparator
- `turn_end` → TurnEndSummary
- `system` → SystemBubble
- `input` → existing user input bubble
- `status` / `error` → existing status/error rendering

### Task 3.2: Update AgentDetailPage inline terminal
- Replace duplicate `TerminalBubble` with shared components from Phase 2
- Ensure same message type dispatch logic

### Task 3.3: Add CSS styles
- Style for ToolUseBubble: tool name badge, collapsible container
- Style for ThinkingBubble: muted background, collapsible
- Style for TurnSeparator: subtle horizontal rule
- Style for TurnEndSummary: small muted text
- Style for SystemBubble: severity colors

### Task 3.4: End-to-end verification
- Test with live backend sending structured messages
- Verify all message types render correctly
- Verify collapsible sections work
- Verify backward compat with old `output` messages
- Verify reconnection replays structured messages correctly
