# Implementation Plan: Virtualized Windowed List Component (Frontend)

**Track ID:** virtual-list-fe_20260310050002Z

## Phase 1: Core VirtualList Component (Foundation)

### Task 1.1: Install @tanstack/react-virtual
- [x] `npm install @tanstack/react-virtual`
- [x] Verify compatibility with React 19 and Vite build
- [x] Confirm bundle size is acceptable

### Task 1.2: Create VirtualList component (TDD)
- [x] Write tests first for:
  - Renders only visible items (not all items)
  - Supports custom renderItem function
  - Applies className to container
  - Handles empty items array
- [x] Create `src/components/virtual/VirtualList.tsx` + CSS module
- [x] Wrap `useVirtualizer` from @tanstack/react-virtual
- [x] Implement Props interface: `items`, `estimateSize`, `renderItem`, `overscan`, `className`
- [x] Expose `VirtualListRef` with `scrollToIndex`

### Task 1.3: Add variable-height support
- [x] Enable `measureElement` for dynamic row measurement
- [x] Use `ResizeObserver` integration from @tanstack/react-virtual
- [x] Test with items of varying heights (short text, long text, code blocks)
- [x] Verify no layout jumps when items are measured

### Task 1.4: Verify Phase 1
- [x] Tests passing (148/148)
- [x] Component renders correctly with mock data of varying sizes
- [x] Build clean (tsc --noEmit passes)

## Phase 2: Auto-Follow and Scroll APIs (Chat Support)

### Task 2.1: Implement auto-follow behavior (TDD)
- [x] Write tests:
  - Auto-scrolls to bottom when new items added and `autoFollow=true`
  - Pauses auto-follow when user scrolls up
  - Resumes auto-follow when user scrolls back to bottom
- [x] Track scroll position relative to bottom
- [x] Use `onScrollStateChange` callback to notify parent of at-bottom state

### Task 2.2: Add scroll-to-bottom indicator
- [x] Floating button that appears when not following (user scrolled up)
- [x] Click to scroll to bottom and resume auto-follow
- [x] Badge showing count of new items since scroll-away
- [x] Style with CSS module matching dashboard theme

### Task 2.3: Implement scrollToIndex API
- [x] Expose via ref: `scrollToIndex(index, { align: 'start' | 'center' | 'end' })`
- [x] Used by diff view for jump-to-file navigation

### Task 2.4: Verify Phase 2
- [x] Auto-follow works correctly in all scenarios
- [x] Scroll-to-bottom indicator appears/disappears correctly
- [x] scrollToIndex navigates accurately
- [x] Tests passing (13 VirtualList tests)

## Phase 3: Performance Validation and Documentation

### Task 3.1: Performance stress test
- [x] Create a test scenario with 10,000+ items of variable height
- [x] Verify DOM node count stays bounded (~10 nodes for 10,000 items)
- [x] Variable-height items render correctly via estimateSize

### Task 3.2: Integration example with terminal messages
- [x] Create integration test rendering terminal-style messages through VirtualList
- [x] Verified heterogeneous message types (text, tool_use, thinking, input)
- [x] VirtualList works with variable-height chat messages

### Task 3.3: Verify Phase 3
- [x] All tests passing (154 total, 16 VirtualList)
- [x] Build clean (`make build` passes)
- [x] `npm run build` succeeds

---

**Total: 10 tasks across 3 phases**
