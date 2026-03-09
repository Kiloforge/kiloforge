# Specification: Virtualized Windowed List Component (Frontend)

**Track ID:** virtual-list-fe_20260310050002Z
**Type:** Feature
**Created:** 2026-03-10T05:00:02Z
**Status:** Draft

## Summary

Build a high-performance virtualized/windowed list component that only renders visible rows plus a small overscan buffer, reusing DOM elements as the user scrolls. This is foundational infrastructure for the agent terminal chat view and diff viewer, both of which can contain thousands of lines of content.

## Context

The dashboard has two views that will grow to contain large amounts of content:

1. **Agent terminal** — chat messages accumulate during long agent sessions (hundreds of turns, each with tool calls, thinking blocks, and text output)
2. **Diff viewer** — large diffs can contain thousands of changed lines across many files

Without virtualization, both views will degrade in performance as content grows — DOM node count increases linearly, layout/paint costs compound, and memory usage grows unbounded. A virtualized list renders only the ~20-30 visible items plus a small buffer, recycling DOM nodes as the user scrolls.

## Codebase Analysis

- **No virtualization library** currently in use — `package.json` has no `react-window`, `react-virtualized`, `@tanstack/react-virtual`, or similar
- **AgentTerminal** uses a simple `messages.map()` rendering all messages — will not scale to 1000+ messages
- **TerminalBubbles** renders heterogeneous message types (text, tool_use, thinking, system) — variable height rows
- **CSS Modules** with custom properties — the component must integrate with the existing styling approach
- **React 19** — must use modern React patterns (refs, hooks)

## Acceptance Criteria

- [ ] `VirtualList` component that accepts items and a render function, only rendering visible items
- [ ] Supports variable-height rows (critical for chat messages and diff hunks of varying sizes)
- [ ] Overscan buffer of configurable rows above/below viewport (default: 5)
- [ ] DOM element recycling — reuses elements rather than creating/destroying on scroll
- [ ] Smooth scrolling with no visible flickering or blank areas during fast scroll
- [ ] `scrollToBottom()` API for auto-scroll in chat view (follow mode)
- [ ] `scrollToIndex(n)` API for jump-to-file in diff view
- [ ] Performance: handles 10,000+ items at 60fps scroll
- [ ] Works with both fixed-height and variable-height items
- [ ] Exposed as a generic, reusable component (not tied to terminal or diff)
- [ ] Unit tests for scroll math and visibility calculations
- [ ] Integration test demonstrating use with terminal messages

## Dependencies

None — this is foundational infrastructure

## Blockers

- branch-diff-view-fe_20260310050001Z should consume this component for its file diff list
- The agent terminal should be migrated to use this component (can be a follow-up track)

## Conflict Risk

None — new component, no file overlap

## Out of Scope

- Migrating existing AgentTerminal to use VirtualList (follow-up track)
- Migrating diff viewer to use VirtualList (can be integrated during diff view implementation)
- Horizontal virtualization (not needed for our use cases)
- Grid/table virtualization (list only)

## Technical Notes

### Approach: Build vs Buy

**Option A — @tanstack/react-virtual** (recommended if acceptable):
- Mature, well-tested, small bundle (~3KB gzipped)
- Supports variable-height items natively
- Headless (no DOM opinions) — fits CSS Modules approach
- Active maintenance, React 19 compatible

**Option B — Custom implementation:**
- Full control over recycling behavior
- No external dependency
- More work but educational and tailored to exact needs

**Recommendation:** Start with `@tanstack/react-virtual` as the engine, wrap it in a `VirtualList` component that provides the project-specific API (scrollToBottom, auto-follow, theme integration). This gets proven scroll math without reinventing it, while keeping the public API clean.

### Component API

```typescript
interface VirtualListProps<T> {
  items: T[];
  estimateSize: (index: number) => number;  // height estimate for unmeasured items
  renderItem: (item: T, index: number) => React.ReactNode;
  overscan?: number;                         // default: 5
  autoFollow?: boolean;                      // auto-scroll to bottom on new items
  className?: string;
  onScrollStateChange?: (atBottom: boolean) => void;
}

interface VirtualListRef {
  scrollToBottom: () => void;
  scrollToIndex: (index: number) => void;
}
```

### Auto-Follow Behavior

For the chat view:
- When `autoFollow` is true and user is scrolled to bottom, new items auto-scroll
- If user scrolls up (not at bottom), auto-follow pauses
- A "scroll to bottom" indicator appears when not following
- Resuming follow when user scrolls back to bottom

### Variable Height Measurement

- Use `ResizeObserver` (via @tanstack/react-virtual's `measureElement`) to measure actual rendered height
- `estimateSize` provides initial height guess to prevent layout jumps
- Once measured, actual height is cached until item changes

---

_Generated by kf-architect from prompt: "Create a highly performant windowed list component for chat view and diffs"_
