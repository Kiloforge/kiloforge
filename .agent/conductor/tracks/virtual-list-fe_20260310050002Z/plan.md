# Implementation Plan: Virtualized Windowed List Component (Frontend)

**Track ID:** virtual-list-fe_20260310050002Z

## Phase 1: Core VirtualList Component (Foundation)

### Task 1.1: Install @tanstack/react-virtual
- `npm install @tanstack/react-virtual`
- Verify compatibility with React 19 and Vite build
- Confirm bundle size is acceptable

### Task 1.2: Create VirtualList component (TDD)
- Write tests first for:
  - Renders only visible items (not all items)
  - Supports custom renderItem function
  - Applies className to container
  - Handles empty items array
- Create `src/components/virtual/VirtualList.tsx` + CSS module
- Wrap `useVirtualizer` from @tanstack/react-virtual
- Implement Props interface: `items`, `estimateSize`, `renderItem`, `overscan`, `className`
- Expose `VirtualListRef` with `scrollToIndex`

### Task 1.3: Add variable-height support
- Enable `measureElement` for dynamic row measurement
- Use `ResizeObserver` integration from @tanstack/react-virtual
- Test with items of varying heights (short text, long text, code blocks)
- Verify no layout jumps when items are measured

### Task 1.4: Verify Phase 1
- Tests passing
- Component renders correctly with mock data of varying sizes
- Build clean

## Phase 2: Auto-Follow and Scroll APIs (Chat Support)

### Task 2.1: Implement auto-follow behavior (TDD)
- Write tests:
  - Auto-scrolls to bottom when new items added and `autoFollow=true`
  - Pauses auto-follow when user scrolls up
  - Resumes auto-follow when user scrolls back to bottom
- Track scroll position relative to bottom
- Use `onScrollStateChange` callback to notify parent of at-bottom state

### Task 2.2: Add scroll-to-bottom indicator
- Floating button that appears when not following (user scrolled up)
- Click to scroll to bottom and resume auto-follow
- Badge showing count of new items since scroll-away
- Style with CSS module matching dashboard theme

### Task 2.3: Implement scrollToIndex API
- Expose via ref: `scrollToIndex(index, { align: 'start' | 'center' | 'end' })`
- Smooth scroll animation option
- Used by diff view for jump-to-file navigation

### Task 2.4: Verify Phase 2
- Auto-follow works correctly in all scenarios
- Scroll-to-bottom indicator appears/disappears correctly
- scrollToIndex navigates accurately
- Tests passing

## Phase 3: Performance Validation and Documentation

### Task 3.1: Performance stress test
- Create a test/storybook scenario with 10,000+ items of variable height
- Verify 60fps scroll performance (no jank)
- Measure DOM node count stays bounded (~30-50 nodes regardless of item count)
- Profile memory usage — verify no leaks on item addition

### Task 3.2: Integration example with terminal messages
- Create a demo/test that renders terminal-style messages through VirtualList
- Verify TerminalBubbles components work correctly inside VirtualList
- Document any caveats (e.g., message grouping, turn separators)

### Task 3.3: Verify Phase 3
- All tests passing
- Performance benchmarks documented
- Build clean
- `npm run build` succeeds

---

**Total: 10 tasks across 3 phases**
