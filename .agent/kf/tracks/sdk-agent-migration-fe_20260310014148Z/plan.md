# Implementation Plan: Structured Agent Terminal Display (Frontend)

**Track ID:** sdk-agent-migration-fe_20260310014148Z

## Phase 1: WebSocket Hook Update

### Task 1.1: Extend WSMessage type
- [x] Add new fields to `WSMessage`: `turnId`, `toolName`, `toolId`, `toolInput`, `thinking`, `costUsd`, `usage`, `subtype`, `data`
- [x] Add new message type literals to the type union

### Task 1.2: Update message handler in useAgentWebSocket
- [x] Add cases for `turn_start`, `text`, `tool_use`, `thinking`, `turn_end`, `system` in the message handler switch
- [x] Map legacy `output` messages to `text` type for backward compatibility
- [x] Parse structured fields from server JSON into `WSMessage`

### Task 1.3: Verify hook handles all message types
- [x] Manual test with mock WS messages
- [x] Ensure backward compat: old `output` messages still render

## Phase 2: Terminal Component Refactor

### Task 2.1: Extract shared terminal components
- [x] Create `frontend/src/components/terminal/` directory
- [x] Extract `MessageBubble` from `AgentTerminal.tsx` into shared component
- [x] Extract `formatCode` utility into shared module
- [x] Update `AgentTerminal.tsx` and `AgentDetailPage.tsx` to import shared components

### Task 2.2: Create ToolUseBubble component
- [x] Display tool name as a styled badge/chip
- [x] Show tool ID in muted text
- [x] Collapsible JSON input display (collapsed by default)
- [x] Use monospace font for JSON, syntax-style colors

### Task 2.3: Create ThinkingBubble component
- [x] Distinct visual style: muted background, italic text
- [x] Collapsible (collapsed by default for long thinking blocks)
- [x] "Thinking..." label with brain/thought icon or indicator

### Task 2.4: Create TurnSeparator component
- [x] Horizontal rule with turn number centered
- [x] Subtle styling that doesn't dominate the terminal

### Task 2.5: Create TurnEndSummary component
- [x] Inline summary below turn separator
- [x] Show: tokens used (input/output), cost in USD
- [x] Muted/small text, right-aligned or centered

### Task 2.6: Create SystemBubble component
- [x] Severity-based styling: warning (yellow), error (red), info (blue)
- [x] Subtype label + message text
- [x] Appropriate icon per severity

## Phase 3: Integration

### Task 3.1: Update AgentTerminal message rendering
- [x] Replace single `MessageBubble` switch with component dispatch based on message type
- [x] `text` / `output` → TextBubble (existing)
- [x] `tool_use` → ToolUseBubble
- [x] `thinking` → ThinkingBubble
- [x] `turn_start` → TurnSeparator
- [x] `turn_end` → TurnEndSummary
- [x] `system` → SystemBubble
- [x] `input` → existing user input bubble
- [x] `status` / `error` → existing status/error rendering

### Task 3.2: Update AgentDetailPage inline terminal
- [x] Replace duplicate `TerminalBubble` with shared components from Phase 2
- [x] Ensure same message type dispatch logic

### Task 3.3: Add CSS styles
- [x] Style for ToolUseBubble: tool name badge, collapsible container
- [x] Style for ThinkingBubble: muted background, collapsible
- [x] Style for TurnSeparator: subtle horizontal rule
- [x] Style for TurnEndSummary: small muted text
- [x] Style for SystemBubble: severity colors

### Task 3.4: End-to-end verification
- [x] Test with live backend sending structured messages
- [x] Verify all message types render correctly
- [x] Verify collapsible sections work
- [x] Verify backward compat with old `output` messages
- [x] Verify reconnection replays structured messages correctly
