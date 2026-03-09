# Implementation Plan: E2E Tests — Interactive Agent Terminal via WebSocket

**Track ID:** e2e-interactive-terminal_20260309194835Z

## Phase 1: Basic Terminal Tests

- [ ] Task 1.1: Spawn interactive agent — click spawn with interactive role, verify terminal UI component appears with a connected indicator
- [ ] Task 1.2: Terminal renders output — verify mock agent's init event and initial output text appear in the terminal display area
- [ ] Task 1.3: Basic input/output — type text in terminal input, press enter, verify mock agent echoes it back as output in the terminal

## Phase 2: Stream-JSON Parsing Tests

- [ ] Task 2.1: Text extraction from events — verify that `content_block_delta` text_delta content is correctly extracted and displayed as plain text in the terminal
- [ ] Task 2.2: Multi-line output — send input that triggers multi-line mock agent response, verify all lines render in correct order with line breaks
- [ ] Task 2.3: Non-text events ignored — verify that `init` and `result` events do not produce spurious text in the terminal output area (status shown separately)

## Phase 3: Reconnection Tests

- [ ] Task 3.1: Disconnect and reconnect — close the WebSocket connection (navigate away and back, or programmatic close), verify terminal reconnects and shows connected indicator
- [ ] Task 3.2: Buffer replay on reconnect — accumulate output from several inputs, disconnect, reconnect, verify all previous output lines (up to 500) are replayed in order
- [ ] Task 3.3: Status sync on reconnect — disconnect while agent is running, reconnect, verify current agent status is correctly displayed (running/completed/failed)

## Phase 4: Multi-Client Tests

- [ ] Task 4.1: Second tab observes output — open a second browser context connected to the same agent, verify it receives the same output stream
- [ ] Task 4.2: Primary client stdin control — verify the first-connected client can type and send input successfully
- [ ] Task 4.3: Second tab cannot send input — verify that the second browser context's input is rejected or input field is disabled/read-only

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Rapid input — send many input lines in quick succession, verify all are processed and echoed back without loss or duplication
- [ ] Task 5.2: Long output lines — trigger mock agent to emit very long lines (>1000 chars), verify terminal handles them without breaking layout
- [ ] Task 5.3: Unicode and special characters — send Unicode text (emoji, CJK, RTL), verify terminal displays them correctly without corruption
- [ ] Task 5.4: Agent crash mid-session — configure mock to crash after N events (`MOCK_AGENT_FAIL_AFTER`), verify terminal shows error/status change and handles gracefully
- [ ] Task 5.5: Connect to nonexistent agent — attempt WebSocket connection to a fake agent ID, verify appropriate error message is displayed
