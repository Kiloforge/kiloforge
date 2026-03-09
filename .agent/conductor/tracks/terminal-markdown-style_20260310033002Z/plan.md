# Implementation Plan: Agent Terminal Markdown Rendering and Terminal-Style Design

**Track ID:** terminal-markdown-style_20260310033002Z

## Phase 1: Markdown Rendering

### Task 1.1: Add react-markdown dependency
- `npm install react-markdown`
- Add to package.json

### Task 1.2: Create MarkdownContent component
- Wrap `react-markdown` with custom renderers for:
  - Code blocks: monospace, dark bg, padding, optional copy button
  - Inline code: monospace with subtle background
  - Links: open in new tab (`target="_blank"`)
  - Lists, bold, italic: standard styling
- Sanitize output (react-markdown handles this by default)

### Task 1.3: Integrate into text message rendering
- Replace raw text display in the text message component with `<MarkdownContent>`
- Ensure thinking blocks also render markdown
- Tool use input/output can remain as JSON

### Task 1.4: Verify Phase 1
- TypeScript compiles
- Markdown renders correctly in terminal (code blocks, lists, bold, links)

## Phase 2: Terminal-Style Visual Redesign

### Task 2.1: Restyle terminal messages area
- Dark background for the messages container
- Monospace font for all message text
- Adjust text color for readability on dark bg (light gray/green)
- Subtle borders or spacing between messages

### Task 2.2: Restyle terminal input area
- Dark input background matching messages area
- Monospace font in textarea
- Subtle border, muted placeholder text
- Send button styled to match

### Task 2.3: Restyle terminal header
- Match the dark theme or provide clear contrast
- Keep connection status dot and agent name visible
- Ensure close button is visible

### Task 2.4: Preserve existing badges
- Verify turn_start, turn_end, tool_use, thinking badges retain their colored chip design
- Adjust badge contrast if needed on dark background (may need slightly higher opacity)

### Task 2.5: Verify Phase 2
- Visual inspection of both AgentTerminal modal and AgentDetailPage terminal
- Badges are still readable and color-coded
- Input area is usable and matches aesthetic

## Phase 3: Polish

### Task 3.1: Code block copy button
- Add a small "Copy" button in the top-right corner of code blocks
- Copies code content to clipboard
- Brief "Copied!" feedback

### Task 3.2: Apply to both terminal contexts
- Ensure AgentTerminal (modal) and AgentDetailPage TerminalSection share the same styles
- Extract shared CSS module if not already shared

### Task 3.3: Verify Phase 3
- TypeScript compiles
- Both terminal contexts look consistent
- Copy button works
